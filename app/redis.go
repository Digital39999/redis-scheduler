package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

var rdb *redis.Client

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

		if len(msg.Payload) < 12 || !strings.HasPrefix(msg.Payload, "rsch-ref:") {
			continue
		}

		go handleExpiration(msg.Payload)
	}
}

func ensureReferences() {
	mainKeysPattern := "rsch:*:*"

	mainKeys, err := rdb.Keys(ctx, mainKeysPattern).Result()
	if err != nil {
		fmt.Println("Failed to retrieve main keys:", err)
		return
	}

	for _, mainKey := range mainKeys {
		scheduleType := strings.Split(mainKey, ":")[1]
		uniqueKey := strings.Split(mainKey, ":")[2]
		refKey := fmt.Sprintf("rsch-ref:%s:%s", scheduleType, uniqueKey)

		refExists, err := rdb.Exists(ctx, refKey).Result()
		if err != nil {
			fmt.Println("Failed to check if reference exists:", err)
			continue
		}

		if refExists == 0 {
			_, err := rdb.Set(ctx, refKey, "", 1*time.Second).Result()
			if err != nil {
				fmt.Printf("Failed to create missing reference for key %s: %v\n", refKey, err)
			} else {
				fmt.Printf("Created missing reference for key %s.\n", refKey)
			}
		}
	}
}

func handleExpiration(refKey string) {
	parts := strings.SplitN(refKey, ":", 3)
	if len(parts) < 3 {
		fmt.Println("Invalid reference key format:", refKey)
		return
	}

	scheduleType := parts[1]
	uniqueKey := parts[2]

	fullKey := fmt.Sprintf("rsch:%s:%s", scheduleType, uniqueKey)

	val, err := rdb.Get(ctx, fullKey).Result()
	if err != nil {
		fmt.Printf("Failed to retrieve unique key data for key %s: %v\n", fullKey, err)
		return
	}

	var reqData RequestData
	if err := json.Unmarshal([]byte(val), &reqData); err != nil {
		fmt.Println("Failed to unmarshal unique key data for key", fullKey, ":", err)
		return
	}

	maxRetries, _ := strconv.Atoi(os.Getenv("RETRIES"))
	retryTime, _ := strconv.Atoi(os.Getenv("RETRY_TIME"))

	if maxRetries != -1 && reqData.Retry >= maxRetries {
		fmt.Printf("Max retries reached for key %s, deleting unique key.\n", fullKey)

		rdb.Del(ctx, fullKey)
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
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
		fmt.Printf("Webhook post failed (%d/%d), retrying in %d seconds.\n", reqData.Retry+1, maxRetries, retryTime)
		reqData.Retry++

		value, err := json.Marshal(reqData)
		if err != nil {
			fmt.Println("Failed to marshal updated request data:", err)
			return
		}

		err = rdb.Set(ctx, fullKey, value, 0).Err()
		if err != nil {
			fmt.Println("Failed to update unique key in Redis:", err)
			return
		}

		err = rdb.Set(ctx, refKey, "", time.Duration(retryTime)*time.Second).Err()
		if err != nil {
			fmt.Println("Failed to reset reference key in Redis:", err)
			return
		}
	} else {
		fmt.Printf("Webhook post successful for key %s, deleting unique key.\n", fullKey)
		rdb.Del(ctx, fullKey)
	}
}
