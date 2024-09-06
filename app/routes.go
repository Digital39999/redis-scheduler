package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func initializeRoutes(router *gin.Engine) {
	router.GET("/", infoHandler)
	router.GET("/stats", statsHandler)

	router.GET("/schedules", allSchedulesHandler)
	router.DELETE("/schedules", deleteAllSchedules)

	router.POST("/schedule", createScheduleHandler)
	router.GET("/schedule/:key", getScheduleHandler)
	router.PATCH("/schedule/:key", patchScheduleHandler)
	router.DELETE("/schedule/:key", deleteScheduleHandler)

	router.NoRoute(notFoundHandler)
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/" {
			c.Next()
			return
		}

		apiAuth := os.Getenv("API_AUTH")
		authHeader := c.GetHeader("Authorization")

		if authHeader != apiAuth {
			c.JSON(http.StatusUnauthorized, gin.H{"status": http.StatusUnauthorized, "data": "Unauthorized."})
			c.Abort()
			return
		}

		c.Next()
	}
}
