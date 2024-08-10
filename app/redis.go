package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
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

	if maxRetries != -1 && reqData.Retry >= maxRetries {
		message := fmt.Sprintf("Max retries reached for key %s, deleting unique key.", uniqueKey)
		fmt.Println(message)

		rdb.Del(ctx, uniqueKey)
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
		fmt.Printf("Webhook post failed (attempt %d/%d), retrying in %d seconds.\n", reqData.Retry+1, maxRetries, retryTime)
		reqData.Retry++

		value, err := json.Marshal(reqData)
		if err != nil {
			fmt.Println("Failed to marshal updated request data:", err)
			return
		}

		err = rdb.Set(ctx, uniqueKey, value, 0).Err()
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
		fmt.Printf("Webhook post successful for key %s, deleting unique key.\n", uniqueKey)
		rdb.Del(ctx, uniqueKey)
	}
}
