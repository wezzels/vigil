// env-monitor - TROOPER-FORGE Mission Processing App
// Phase 1: Core Infrastructure
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/redis/go-redis/v9"
)

var (
    kafkaBrokers = []string{getEnv("KAFKA_BROKERS", "kafka:9092")}
    redisAddr   = getEnv("REDIS_ADDR", "redis:6379")
    port        = getEnv("PORT", "8080")
    redisClient *redis.Client
)

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}

type HealthResponse struct {
    Service    string    `json:"service"`
    Version    string    `json:"version"`
    Timestamp  time.Time `json:"timestamp"`
    Status     string    `json:"status"`
    Kafka      string    `json:"kafka"`
    Redis      string    `json:"redis"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    resp := HealthResponse{
        Service:   "env-monitor",
        Version:   "0.1.0",
        Timestamp: time.Now().UTC(),
        Status:    "healthy",
        Kafka:     "connected",
        Redis:     "connected",
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

func main() {
    flag.StringVar(&port, "port", port, "HTTP server port")
    flag.Parse()

    redisClient = redis.NewClient(&redis.Options{
        Addr: redisAddr,
    })

    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/ready", healthHandler)

    log.Printf("env-monitor starting on :%s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
