package main

import (
	"context"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var ctx = context.Background()

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

	router.Use(authMiddleware())
	initializeRoutes(router)

	db := 0
	if rdb.Options().DB != 0 {
		db = rdb.Options().DB
	}

	go listenForExpirations(db)

	if err := router.Run(":" + port); err != nil {
		fmt.Println("Failed to run server:", err)
	}
}
