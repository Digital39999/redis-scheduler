package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func initializeRoutes(router *gin.Engine) {
	router.POST("/schedule", scheduleHandler)
	router.POST("/purge", purgeHandler)
	router.GET("/stats", statsHandler)
	router.GET("/", infoHandler)

	router.GET("/schedules", schedulesHandler)
	router.GET("/schedules/:key", getScheduleHandler)
	router.PATCH("/schedules/:key", patchScheduleHandler)
	router.DELETE("/schedules/:key", deleteScheduleHandler)

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
