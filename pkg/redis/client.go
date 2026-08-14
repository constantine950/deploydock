package redis

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

// NewClient creates a Redis client from REDIS_URL env var.
func NewClient() (*redis.Client, error) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379"
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_URL: %w", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return client, nil
}

// LogChannel returns the Redis pub/sub channel name for a deployment.
func LogChannel(deploymentID string) string {
	return "logs:" + deploymentID
}

// PublishLog publishes a log line to a deployment's pub/sub channel.
func PublishLog(ctx context.Context, rdb *redis.Client, deploymentID, line string) {
	rdb.Publish(ctx, LogChannel(deploymentID), line)
}