// Package config provides Redis configuration and connection management.
package config

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds Redis connection configuration.
type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
}

// RedisClient wraps redis.Client with configuration.
type RedisClient struct {
	*redis.Client
	Config RedisConfig
}

// DefaultRedisConfig returns the default Redis configuration.
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 10,
	}
}

// NewRedisClient creates a new Redis client with the given configuration.
func NewRedisClient(cfg RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		Client: client,
		Config: cfg,
	}, nil
}

// NewRedisClientWithoutPing creates a new Redis client without verifying connection.
// Useful for testing with miniredis or when Redis is not yet available.
func NewRedisClientWithoutPing(cfg RedisConfig) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	return &RedisClient{
		Client: client,
		Config: cfg,
	}
}

// Close closes the Redis client connection.
func (r *RedisClient) Close() error {
	return r.Client.Close()
}

// Ping verifies the Redis connection is still alive.
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}
