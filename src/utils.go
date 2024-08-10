package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/cpu"
)

var startTime = time.Now()

func loadEnvVars() (string, string, error) {
	redisURL := os.Getenv("REDIS_URL")
	port := os.Getenv("PORT")

	if redisURL == "" || port == "" {
		return "", "", errors.New("missing REDIS_URL or PORT in environment variables")
	}

	if os.Getenv("API_AUTH") == "" {
		return "", "", errors.New("missing API_AUTH in environment variables")
	}

	if os.Getenv("RETRY_TIME") == "" || os.Getenv("RETRIES") == "" {
		return "", "", errors.New("missing RETRY_TIME or RETRIES in environment variables")
	}

	return redisURL, port, nil
}

func generateRandomKey() (string, error) {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func getCpuUsage() float64 {
	percent, err := cpu.Percent(0, false)
	if err != nil {
		return 0
	}

	if len(percent) > 0 {
		return math.Round(percent[0]*100) / 100
	}

	return 0
}

func getMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	totalAllocated := m.Alloc + m.TotalAlloc
	return totalAllocated
}

func formatBytes(bytes uint64) string {
	const (
		_         = iota
		KB uint64 = 1 << (10 * iota)
		MB
		GB
		TB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2fTB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2fGB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
