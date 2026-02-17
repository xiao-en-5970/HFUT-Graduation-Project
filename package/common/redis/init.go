package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
)

var Client *redis.Client
var Ctx = context.Background()

// Init initializes Redis connection
func Init() error {
	db, err := strconv.Atoi(config.RedisDB)
	if err != nil {
		return fmt.Errorf("invalid Redis DB number: %w", err)
	}

	Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
		Password: config.RedisPassword,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = Client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return nil
}

// Close closes the Redis connection
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}
