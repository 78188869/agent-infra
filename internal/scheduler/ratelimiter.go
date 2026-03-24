package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// TenantQuotaKeyPattern is the pattern for tenant quota usage keys.
	TenantQuotaKeyPattern = "scheduler:tenant:%s:quota"
	// GlobalQuotaKey is the key for global quota usage.
	GlobalQuotaKey = "scheduler:global:quota"
	// TenantDailyKeyPattern is the pattern for tenant daily task count keys.
	TenantDailyKeyPattern = "scheduler:tenant:%s:daily:%s"
)

// QuotaUsage represents current quota usage.
type QuotaUsage struct {
	CurrentConcurrency int `json:"current_concurrency"`
	TodayTasks         int `json:"today_tasks"`
}

// TenantQuota represents tenant quota configuration.
type TenantQuota struct {
	Concurrency int
	DailyTasks  int
}

// RateLimiter manages rate limiting for tenant and global quotas.
type RateLimiter struct {
	client      *redis.Client
	globalLimit int
}

// NewRateLimiter creates a new RateLimiter instance.
func NewRateLimiter(client *redis.Client, globalLimit int) *RateLimiter {
	return &RateLimiter{
		client:      client,
		globalLimit: globalLimit,
	}
}

// Allow checks if a task can be scheduled based on quota limits.
// Returns an error if any limit is exceeded.
func (r *RateLimiter) Allow(ctx context.Context, tenantID string, quota *TenantQuota) error {
	// Check tenant concurrency
	usage, err := r.GetUsage(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant usage: %w", err)
	}

	if usage.CurrentConcurrency >= quota.Concurrency {
		return ErrQuotaExceeded
	}

	// Check daily task limit
	if usage.TodayTasks >= quota.DailyTasks {
		return ErrDailyLimitExceeded
	}

	// Check global concurrency limit
	globalUsage, err := r.getGlobalUsage(ctx)
	if err != nil {
		return fmt.Errorf("failed to get global usage: %w", err)
	}

	if globalUsage.CurrentConcurrency >= r.globalLimit {
		return ErrGlobalLimitExceeded
	}

	return nil
}

// Reserve increments the concurrency counters for tenant and global.
// This should be called when a task starts execution.
func (r *RateLimiter) Reserve(ctx context.Context, tenantID string) error {
	// Increment tenant concurrency counter
	key := fmt.Sprintf(TenantQuotaKeyPattern, tenantID)
	err := r.client.HIncrBy(ctx, key, "current_concurrency", 1).Err()
	if err != nil {
		return fmt.Errorf("failed to reserve tenant concurrency: %w", err)
	}

	// Increment daily task counter
	today := time.Now().Format("2006-01-02")
	dailyKey := fmt.Sprintf(TenantDailyKeyPattern, tenantID, today)
	if err := r.client.Incr(ctx, dailyKey).Err(); err != nil {
		return fmt.Errorf("failed to increment daily tasks: %w", err)
	}
	// Set expiration for daily key (expire at end of day)
	r.client.Expire(ctx, dailyKey, 24*time.Hour)

	// Increment global counter
	if err := r.client.HIncrBy(ctx, GlobalQuotaKey, "current_concurrency", 1).Err(); err != nil {
		return fmt.Errorf("failed to reserve global concurrency: %w", err)
	}

	return nil
}

// Release decrements the concurrency counters for tenant and global.
// This should be called when a task completes or is preempted.
func (r *RateLimiter) Release(ctx context.Context, tenantID string) error {
	// Decrement tenant concurrency counter
	key := fmt.Sprintf(TenantQuotaKeyPattern, tenantID)
	result, err := r.client.HGet(ctx, key, "current_concurrency").Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get tenant concurrency: %w", err)
	}

	// Only decrement if counter is positive
	if result != "" {
		val, _ := strconv.Atoi(result)
		if val > 0 {
			if err := r.client.HIncrBy(ctx, key, "current_concurrency", -1).Err(); err != nil {
				return fmt.Errorf("failed to release tenant concurrency: %w", err)
			}
		}
	}

	// Decrement global counter
	globalResult, err := r.client.HGet(ctx, GlobalQuotaKey, "current_concurrency").Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get global concurrency: %w", err)
	}

	if globalResult != "" {
		val, _ := strconv.Atoi(globalResult)
		if val > 0 {
			if err := r.client.HIncrBy(ctx, GlobalQuotaKey, "current_concurrency", -1).Err(); err != nil {
				return fmt.Errorf("failed to release global concurrency: %w", err)
			}
		}
	}

	return nil
}

// GetUsage returns the current quota usage for a tenant.
func (r *RateLimiter) GetUsage(ctx context.Context, tenantID string) (*QuotaUsage, error) {
	key := fmt.Sprintf(TenantQuotaKeyPattern, tenantID)
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant quota: %w", err)
	}

	usage := &QuotaUsage{}
	if v, ok := result["current_concurrency"]; ok {
		usage.CurrentConcurrency, _ = strconv.Atoi(v)
	}

	// Get today's task count
	today := time.Now().Format("2006-01-02")
	dailyKey := fmt.Sprintf(TenantDailyKeyPattern, tenantID, today)
	todayTasks, err := r.client.Get(ctx, dailyKey).Result()
	if err == nil {
		usage.TodayTasks, _ = strconv.Atoi(todayTasks)
	} else if err != redis.Nil {
		return nil, fmt.Errorf("failed to get daily tasks: %w", err)
	}

	return usage, nil
}

// getGlobalUsage returns the current global concurrency usage.
func (r *RateLimiter) getGlobalUsage(ctx context.Context) (*QuotaUsage, error) {
	result, err := r.client.HGet(ctx, GlobalQuotaKey, "current_concurrency").Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get global quota: %w", err)
	}

	usage := &QuotaUsage{}
	if result != "" {
		usage.CurrentConcurrency, _ = strconv.Atoi(result)
	}

	return usage, nil
}

// GetGlobalConcurrency returns the current global concurrency count.
func (r *RateLimiter) GetGlobalConcurrency(ctx context.Context) (int, error) {
	usage, err := r.getGlobalUsage(ctx)
	if err != nil {
		return 0, err
	}
	return usage.CurrentConcurrency, nil
}

// Reset resets the quota usage for a tenant (for testing).
func (r *RateLimiter) Reset(ctx context.Context, tenantID string) error {
	key := fmt.Sprintf(TenantQuotaKeyPattern, tenantID)
	return r.client.Del(ctx, key).Err()
}

// ResetGlobal resets the global quota usage (for testing).
func (r *RateLimiter) ResetGlobal(ctx context.Context) error {
	return r.client.Del(ctx, GlobalQuotaKey).Err()
}
