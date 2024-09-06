package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type RequestData struct {
	Webhook string      `json:"webhook" binding:"required"`
	Retry   int         `json:"-"`
	TTL     int         `json:"ttl" binding:"required,min=1"`
	Data    interface{} `json:"data" binding:"required"`
}

type PartialRequestData struct {
	Webhook string      `json:"webhook"`
	Retry   int         `json:"-"`
	TTL     int         `json:"ttl"`
	Data    interface{} `json:"data"`
}

func createScheduleHandler(c *gin.Context) {
	var reqData RequestData
	if err := c.ShouldBindJSON(&reqData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "error": fmt.Sprintf("Invalid input: %v", err.Error())})
		return
	}

	scheduleType := c.Query("type")
	if scheduleType == "" {
		scheduleType = "default"
	}

	uniqueKey, err := generateRandomKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "Failed to generate unique key."})
		return
	}

	fullKey := fmt.Sprintf("rsch:%s:%s", scheduleType, uniqueKey)
	reqData.Retry = 0

	value, err := json.Marshal(reqData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "Failed to marshal request data."})
		return
	}

	err = rdb.Set(ctx, fullKey, value, 0).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": fmt.Sprintf("Failed to save data in Redis: %v", err.Error())})
		return
	}

	refKey := fmt.Sprintf("rsch-ref:%s:%s", scheduleType, uniqueKey)
	err = rdb.Set(ctx, refKey, "", time.Duration(reqData.TTL)*time.Second).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": fmt.Sprintf("Failed to save reference key in Redis: %v", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": refKey})
}

func getScheduleHandler(c *gin.Context) {
	key := c.Param("key")
	scheduleType := "default"
	uniqueKey := key

	if len(key) > len("rsch-ref:") && key[:len("rsch-ref:")] == "rsch-ref:" {
		parts := strings.SplitN(key[len("rsch-ref:"):], ":", 2)
		if len(parts) == 2 {
			scheduleType = parts[0]
			uniqueKey = parts[1]
		}
	}

	fullKey := fmt.Sprintf("rsch:%s:%s", scheduleType, uniqueKey)

	data, err := rdb.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "error": "Schedule not found."})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "Failed to retrieve schedule."})
		return
	}

	var reqData RequestData
	if err := json.Unmarshal([]byte(data), &reqData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "Failed to unmarshal schedule data."})
		return
	}

	schedule := make(map[string]interface{})
	schedule["info"] = map[string]interface{}{
		"key":     key,
		"ttl":     reqData.TTL,
		"retry":   reqData.Retry,
		"webhook": reqData.Webhook,
		"expires": time.Now().Add(time.Duration(reqData.TTL) * time.Second),
	}

	schedule["data"] = reqData.Data
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": schedule})
}

func allSchedulesHandler(c *gin.Context) {
	scheduleType := c.Query("type")
	if scheduleType == "" {
		scheduleType = "*"
	}

	keysPattern := fmt.Sprintf("rsch-ref:%s:*", scheduleType)
	keys, err := rdb.Keys(ctx, keysPattern).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "Failed to retrieve reference keys."})
		return
	}

	schedules := make([]map[string]interface{}, 0)
	for _, key := range keys {
		fullKey := key[len("rsch-ref:"):]
		parts := strings.SplitN(fullKey, ":", 2)
		if len(parts) < 2 {
			continue
		}

		scheduleType := parts[0]
		uniqueKey := parts[1]

		mainKey := fmt.Sprintf("rsch:%s:%s", scheduleType, uniqueKey)
		data, err := rdb.Get(ctx, mainKey).Result()
		if err != nil {
			continue
		}

		var reqData RequestData
		if err := json.Unmarshal([]byte(data), &reqData); err != nil {
			continue
		}

		schedule := make(map[string]interface{})
		schedule["info"] = map[string]interface{}{
			"key":     key,
			"ttl":     reqData.TTL,
			"retry":   reqData.Retry,
			"webhook": reqData.Webhook,
			"expires": time.Now().Add(time.Duration(reqData.TTL) * time.Second),
		}

		schedule["data"] = reqData.Data
		schedules = append(schedules, schedule)
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": schedules})
}

func patchScheduleHandler(c *gin.Context) {
	key := c.Param("key")
	scheduleType := "default"
	uniqueKey := key

	if len(key) > len("rsch-ref:") && key[:len("rsch-ref:")] == "rsch-ref:" {
		parts := strings.SplitN(key[len("rsch-ref:"):], ":", 2)
		if len(parts) == 2 {
			scheduleType = parts[0]
			uniqueKey = parts[1]
		}
	}

	fullKey := fmt.Sprintf("rsch:%s:%s", scheduleType, uniqueKey)

	var reqData PartialRequestData
	if err := c.ShouldBindJSON(&reqData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "error": fmt.Sprintf("Invalid input: %v", err.Error())})
		return
	}

	existingData, err := rdb.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "error": "Schedule not found."})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "Failed to retrieve schedule."})
		return
	}

	var existingRequestData RequestData
	if err := json.Unmarshal([]byte(existingData), &existingRequestData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "Failed to unmarshal existing data."})
		return
	}

	if reqData.Webhook != "" {
		existingRequestData.Webhook = reqData.Webhook
	}

	if reqData.Retry != 0 {
		existingRequestData.Retry = reqData.Retry
	}

	if reqData.TTL > 0 {
		existingRequestData.TTL = reqData.TTL
	}

	if reqData.Data != nil {
		existingRequestData.Data = reqData.Data
	}

	updatedValue, err := json.Marshal(existingRequestData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "Failed to marshal updated request data."})
		return
	}

	err = rdb.Set(ctx, fullKey, updatedValue, 0).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": fmt.Sprintf("Failed to update data in Redis: %v", err.Error())})
		return
	}

	refKey := fmt.Sprintf("rsch-ref:%s:%s", scheduleType, uniqueKey)
	err = rdb.Expire(ctx, refKey, time.Duration(existingRequestData.TTL)*time.Second).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": fmt.Sprintf("Failed to update reference key TTL in Redis: %v", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": "Schedule updated successfully."})
}

func deleteScheduleHandler(c *gin.Context) {
	key := c.Param("key")
	scheduleType := "default"
	uniqueKey := key

	if len(key) > len("rsch-ref:") && key[:len("rsch-ref:")] == "rsch-ref:" {
		parts := strings.SplitN(key[len("rsch-ref:"):], ":", 2)
		if len(parts) == 2 {
			scheduleType = parts[0]
			uniqueKey = parts[1]
		}
	}

	fullKey := fmt.Sprintf("rsch:%s:%s", scheduleType, uniqueKey)

	_, err := rdb.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "error": "Schedule not found."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "Failed to retrieve schedule."})
		return
	}

	err = rdb.Del(ctx, fullKey, fmt.Sprintf("rsch-ref:%s:%s", scheduleType, uniqueKey)).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": fmt.Sprintf("Failed to delete schedule from Redis: %v", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": "Schedule deleted successfully."})
}

func deleteAllSchedules(c *gin.Context) {
	err := rdb.FlushDB(ctx).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "error": "Failed to purge Redis database."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": "Redis database purged successfully."})
}

func statsHandler(c *gin.Context) {
	totalKeys, _ := rdb.DBSize(ctx).Result()
	scheduleKeys, _ := rdb.Keys(ctx, "rsch-ref-*").Result()
	numSchedules := len(scheduleKeys)

	memoryBytes := getMemoryUsage()

	stats := gin.H{
		"total_redis_keys":  totalKeys,
		"running_schedules": numSchedules,

		"cpu_usage": getCpuUsage(),
		"ram_usage": formatBytes(memoryBytes),

		"ram_usage_bytes": memoryBytes,

		"system_uptime": time.Since(startTime).String(),
		"go_routines":   runtime.NumGoroutine(),
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": stats})
}

func infoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": "Scheduler service is running."})
}

func notFoundHandler(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "error": "Route not found."})
}
