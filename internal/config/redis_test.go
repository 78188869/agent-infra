package config

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestDefaultRedisConfig(t *testing.T) {
	cfg := DefaultRedisConfig()
	if cfg.Addr != "localhost:6379" {
		t.Errorf("expected default addr localhost:6379, got %s", cfg.Addr)
	}
	if cfg.PoolSize != 100 {
		t.Errorf("expected default pool size 100, got %d", cfg.PoolSize)
	}
	if cfg.MinIdleConns != 10 {
		t.Errorf("expected default min idle conns 10, got %d", cfg.MinIdleConns)
	}
}

func TestNewRedisClientWithoutPing(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	cfg := RedisConfig{
		Addr:     mr.Addr(),
		PoolSize: 10,
	}
	client := NewRedisClientWithoutPing(cfg)
	if client == nil {
		t.Fatal("expected client, got nil")
	}
	defer client.Close()

	// Verify client works
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Errorf("failed to ping redis: %v", err)
	}
}

func TestNewRedisClient_WithConnection(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	cfg := RedisConfig{
		Addr:     mr.Addr(),
		PoolSize: 10,
	}
	client, err := NewRedisClient(cfg)
	if err != nil {
		t.Fatalf("failed to create redis client: %v", err)
	}
	defer client.Close()

	// Verify connection works
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Errorf("failed to ping redis: %v", err)
	}
}

func TestNewRedisClient_ConnectionFailure(t *testing.T) {
	cfg := RedisConfig{
		Addr:     "localhost:16379", // Non-existent server
		PoolSize: 10,
	}
	_, err := NewRedisClient(cfg)
	if err == nil {
		t.Error("expected error for failed connection, got nil")
	}
}

func TestRedisClient_Close(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	cfg := RedisConfig{
		Addr:     mr.Addr(),
		PoolSize: 10,
	}
	client, err := NewRedisClient(cfg)
	if err != nil {
		t.Fatalf("failed to create redis client: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Errorf("failed to close client: %v", err)
	}
}
