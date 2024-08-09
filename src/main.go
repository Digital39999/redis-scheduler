package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

var ctx = context.Background()
var rdb *redis.Client

type RequestData struct {
	Webhook string      `json:"webhook" binding:"required"`   // Webhook URL
	Retry   int         `json:"retry"`                        // Retry count
	TTL     int         `json:"ttl" binding:"required,min=1"` // Time-to-live in seconds
	Data    interface{} `json:"data" binding:"required"`      // Actual data to be sent
}

func generateRandomKey() (string, error) {
	b := make([]byte, 16) // 16 bytes = 128 bits
	_, err := rand.Read(b)

	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

func scheduleHandler(c *gin.Context) {
	apiAuth := os.Getenv("API_AUTH")
	authHeader := c.GetHeader("Authorization")

	if authHeader != apiAuth {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": http.StatusUnauthorized,
			"data":   "Unauthorized.",
		})

		return
	}

	var reqData RequestData
	if err := c.ShouldBindJSON(&reqData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": http.StatusBadRequest,
			"data":   fmt.Sprintf("Invalid input: %v", err.Error()),
		})

		return
	}

	uniqueKey, err := generateRandomKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": http.StatusInternalServerError,
			"data":   "Failed to generate unique key.",
		})

		return
	}

	uniqueKey = "rsch:unique:" + uniqueKey
	reqData.Retry = 0

	value, err := json.Marshal(reqData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": http.StatusInternalServerError,
			"data":   "Failed to marshal request data.",
		})

		return
	}

	err = rdb.Set(ctx, uniqueKey, value, 0).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": http.StatusInternalServerError,
			"data":   fmt.Sprintf("Failed to save data in Redis: %v", err.Error()),
		})

		return
	}

	refKey := "rsch:ref-" + uniqueKey[len("rsch:unique:"):]
	err = rdb.Set(ctx, refKey, "", time.Duration(reqData.TTL)*time.Second).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": http.StatusInternalServerError,
			"data":   fmt.Sprintf("Failed to save reference key in Redis: %v", err.Error()),
		})

		return
	}

	message := fmt.Sprintf("Scheduled webhook for key %s, TTL: %d seconds.", refKey, reqData.TTL)
	fmt.Println(message)

	c.JSON(http.StatusOK, gin.H{
		"status": http.StatusOK,
		"data":   fmt.Sprintf("Submission successful, key: %s", refKey),
	})
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file.")
	}

	redisURL, port, err := loadEnvVars()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	rdb, err = createRedisClient(redisURL)
	if err != nil {
		fmt.Println("Failed to create Redis client:", err)
		return
	}

	router := gin.Default()
	router.POST("/schedule", scheduleHandler)

	db := 0
	if rdb.Options().DB != 0 {
		db = rdb.Options().DB
	}

	go listenForExpirations(db)

	if err := router.Run(":" + port); err != nil {
		fmt.Println("Failed to run server:", err)
	}
}

func createRedisClient(redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		fmt.Println("Invalid Redis URL:", err)
		return nil, err
	}

	client := redis.NewClient(opt)
	_, err = client.Ping(ctx).Result()
	if err != nil {
		fmt.Println("Failed to connect to Redis:", err)
		return nil, err
	}

	fmt.Println("Connected to Redis.")

	_, err = client.ConfigSet(ctx, "notify-keyspace-events", "Ex").Result()
	if err != nil {
		fmt.Println("Failed to set notify-keyspace-events:", err)
		return nil, err
	}

	return client, nil
}

func loadEnvVars() (string, string, error) {
	redisURL := os.Getenv("REDIS_URL")
	apiAuth := os.Getenv("API_AUTH")
	port := os.Getenv("PORT")
	retries := os.Getenv("RETRIES")
	retryTime := os.Getenv("RETRY_TIME")

	if redisURL == "" {
		return "", "", errors.New("REDIS_URL is not set")
	}

	if apiAuth == "" {
		return "", "", errors.New("API_AUTH is not set")
	}

	if port == "" {
		return "", "", errors.New("PORT is not set")
	}

	if retries == "" {
		return "", "", errors.New("RETRIES is not set")
	}

	if retryTime == "" {
		return "", "", errors.New("RETRY_TIME is not set")
	}

	return redisURL, port, nil
}

func listenForExpirations(db int) {
	eventName := "__keyevent@" + strconv.Itoa(db) + "__:expired"

	pubsub := rdb.PSubscribe(ctx, eventName)
	defer pubsub.Close()

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			fmt.Println("Error subscribing:", err)
			continue
		}

		if len(msg.Payload) < 8 || msg.Payload[:9] != "rsch:ref-" {
			continue
		}

		go handleExpiration(msg.Payload)
	}
}

func handleExpiration(refKey string) {
	uniqueKey := "rsch:unique:" + refKey[len("rsch:ref-"):]

	val, err := rdb.Get(ctx, uniqueKey).Result()
	if err != nil {
		fmt.Println("Failed to get unique key data:", err)
		return
	}

	var reqData RequestData
	if err := json.Unmarshal([]byte(val), &reqData); err != nil {
		fmt.Println("Failed to unmarshal unique key data:", err)
		return
	}

	maxRetries, _ := strconv.Atoi(os.Getenv("RETRIES"))
	retryTime, _ := strconv.Atoi(os.Getenv("RETRY_TIME"))

	if reqData.Retry >= maxRetries {
		message := fmt.Sprintf("Max retries reached for key %s, deleting unique key.", uniqueKey)
		fmt.Println(message)

		rdb.Del(ctx, uniqueKey)
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second, // 10 second timeout
	}

	body, err := json.Marshal(reqData.Data)
	if err != nil {
		fmt.Println("Error marshaling request data:", err)
		return
	}

	req, err := http.NewRequest("POST", reqData.Webhook, bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("Error creating POST request:", err)
		return
	}

	req.Header.Set("Authorization", os.Getenv("API_AUTH"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf("Webhook post failed (attempt %d/%d), retrying in %d seconds.\n", reqData.Retry+1, maxRetries, retryTime)
		reqData.Retry++

		value, err := json.Marshal(reqData)
		if err != nil {
			fmt.Println("Failed to marshal updated request data:", err)
			return
		}

		err = rdb.Set(ctx, uniqueKey, value, 0).Err()
		if err != nil {
			fmt.Println("Failed to update retry count in Redis:", err)
			return
		}

		newRefKey := "rsch:ref-" + uniqueKey[len("rsch:unique:"):]
		err = rdb.Set(ctx, newRefKey, "", time.Duration(retryTime)*time.Second).Err()
		if err != nil {
			fmt.Println("Failed to create new reference key in Redis:", err)
			return
		}

		return // End execution, the retry will be handled by the expiration system
	}

	message := fmt.Sprintf("Webhook post successful for key %s, deleting unique key.", uniqueKey)
	fmt.Println(message)

	rdb.Del(ctx, uniqueKey)
}
