package scheduler

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupTestRateLimiter(t *testing.T, globalLimit int) (*RateLimiter, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	limiter := NewRateLimiter(client, globalLimit)
	return limiter, mr
}

func TestRateLimiter_Allow_WithinLimits(t *testing.T) {
	limiter, mr := setupTestRateLimiter(t, 100)
	defer mr.Close()
	defer limiter.client.Close()

	ctx := context.Background()
	quota := &TenantQuota{
		Concurrency: 10,
		DailyTasks:  100,
	}

	err := limiter.Allow(ctx, "tenant-1", quota)
	if err != nil {
		t.Errorf("expected allow within limits, got error: %v", err)
	}
}

func TestRateLimiter_Allow_TenantConcurrencyExceeded(t *testing.T) {
	limiter, mr := setupTestRateLimiter(t, 100)
	defer mr.Close()
	defer limiter.client.Close()

	ctx := context.Background()
	quota := &TenantQuota{
		Concurrency: 2,
		DailyTasks:  100,
	}

	// Reserve 2 slots
	if err := limiter.Reserve(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to reserve: %v", err)
	}
	if err := limiter.Reserve(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to reserve: %v", err)
	}

	// Should exceed concurrency limit
	err := limiter.Allow(ctx, "tenant-1", quota)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got: %v", err)
	}
}

func TestRateLimiter_Allow_GlobalConcurrencyExceeded(t *testing.T) {
	limiter, mr := setupTestRateLimiter(t, 2)
	defer mr.Close()
	defer limiter.client.Close()

	ctx := context.Background()
	quota := &TenantQuota{
		Concurrency: 100,
		DailyTasks:  100,
	}

	// Reserve 2 global slots from different tenants
	if err := limiter.Reserve(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to reserve: %v", err)
	}
	if err := limiter.Reserve(ctx, "tenant-2"); err != nil {
		t.Fatalf("failed to reserve: %v", err)
	}

	// Should exceed global limit
	err := limiter.Allow(ctx, "tenant-3", quota)
	if err != ErrGlobalLimitExceeded {
		t.Errorf("expected ErrGlobalLimitExceeded, got: %v", err)
	}
}

func TestRateLimiter_Allow_DailyLimitExceeded(t *testing.T) {
	limiter, mr := setupTestRateLimiter(t, 100)
	defer mr.Close()
	defer limiter.client.Close()

	ctx := context.Background()
	quota := &TenantQuota{
		Concurrency: 10,
		DailyTasks:  2,
	}

	// Simulate 2 tasks today
	if err := limiter.Reserve(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to reserve: %v", err)
	}
	if err := limiter.Release(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to release: %v", err)
	}
	if err := limiter.Reserve(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to reserve: %v", err)
	}
	if err := limiter.Release(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to release: %v", err)
	}

	// Should exceed daily limit
	err := limiter.Allow(ctx, "tenant-1", quota)
	if err != ErrDailyLimitExceeded {
		t.Errorf("expected ErrDailyLimitExceeded, got: %v", err)
	}
}

func TestRateLimiter_ReserveAndRelease(t *testing.T) {
	limiter, mr := setupTestRateLimiter(t, 100)
	defer mr.Close()
	defer limiter.client.Close()

	ctx := context.Background()

	// Reserve a slot
	if err := limiter.Reserve(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to reserve: %v", err)
	}

	// Check usage
	usage, err := limiter.GetUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("failed to get usage: %v", err)
	}
	if usage.CurrentConcurrency != 1 {
		t.Errorf("expected concurrency 1, got %d", usage.CurrentConcurrency)
	}

	// Check global concurrency
	globalConc, err := limiter.GetGlobalConcurrency(ctx)
	if err != nil {
		t.Fatalf("failed to get global concurrency: %v", err)
	}
	if globalConc != 1 {
		t.Errorf("expected global concurrency 1, got %d", globalConc)
	}

	// Release the slot
	if err := limiter.Release(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to release: %v", err)
	}

	// Check usage after release
	usage, err = limiter.GetUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("failed to get usage after release: %v", err)
	}
	if usage.CurrentConcurrency != 0 {
		t.Errorf("expected concurrency 0 after release, got %d", usage.CurrentConcurrency)
	}

	// Check global concurrency after release
	globalConc, err = limiter.GetGlobalConcurrency(ctx)
	if err != nil {
		t.Fatalf("failed to get global concurrency: %v", err)
	}
	if globalConc != 0 {
		t.Errorf("expected global concurrency 0 after release, got %d", globalConc)
	}
}

func TestRateLimiter_Release_DoesNotGoNegative(t *testing.T) {
	limiter, mr := setupTestRateLimiter(t, 100)
	defer mr.Close()
	defer limiter.client.Close()

	ctx := context.Background()

	// Release without reserve (should not go negative)
	if err := limiter.Release(ctx, "tenant-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	usage, err := limiter.GetUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("failed to get usage: %v", err)
	}
	if usage.CurrentConcurrency < 0 {
		t.Errorf("concurrency should not be negative, got %d", usage.CurrentConcurrency)
	}
}

func TestRateLimiter_GetUsage_NewTenant(t *testing.T) {
	limiter, mr := setupTestRateLimiter(t, 100)
	defer mr.Close()
	defer limiter.client.Close()

	ctx := context.Background()

	usage, err := limiter.GetUsage(ctx, "new-tenant")
	if err != nil {
		t.Fatalf("failed to get usage for new tenant: %v", err)
	}
	if usage.CurrentConcurrency != 0 {
		t.Errorf("expected 0 concurrency for new tenant, got %d", usage.CurrentConcurrency)
	}
	if usage.TodayTasks != 0 {
		t.Errorf("expected 0 today tasks for new tenant, got %d", usage.TodayTasks)
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	limiter, mr := setupTestRateLimiter(t, 100)
	defer mr.Close()
	defer limiter.client.Close()

	ctx := context.Background()

	// Reserve some slots
	limiter.Reserve(ctx, "tenant-1")
	limiter.Reserve(ctx, "tenant-1")

	// Reset
	if err := limiter.Reset(ctx, "tenant-1"); err != nil {
		t.Fatalf("failed to reset: %v", err)
	}

	// Check usage is reset
	usage, err := limiter.GetUsage(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("failed to get usage after reset: %v", err)
	}
	if usage.CurrentConcurrency != 0 {
		t.Errorf("expected 0 concurrency after reset, got %d", usage.CurrentConcurrency)
	}
}

func TestRateLimiter_MultipleTenants(t *testing.T) {
	limiter, mr := setupTestRateLimiter(t, 10)
	defer mr.Close()
	defer limiter.client.Close()

	ctx := context.Background()
	quota := &TenantQuota{
		Concurrency: 5,
		DailyTasks:  100,
	}

	// Reserve for multiple tenants
	for i := 0; i < 3; i++ {
		tenantID := string(rune('a' + i))
		if err := limiter.Reserve(ctx, tenantID); err != nil {
			t.Fatalf("failed to reserve for %s: %v", tenantID, err)
		}
	}

	// Check global concurrency
	globalConc, err := limiter.GetGlobalConcurrency(ctx)
	if err != nil {
		t.Fatalf("failed to get global concurrency: %v", err)
	}
	if globalConc != 3 {
		t.Errorf("expected global concurrency 3, got %d", globalConc)
	}

	// Each tenant should still be within limits
	for i := 0; i < 3; i++ {
		tenantID := string(rune('a' + i))
		err := limiter.Allow(ctx, tenantID, quota)
		if err != nil {
			t.Errorf("tenant %s should be allowed, got: %v", tenantID, err)
		}
	}
}

func TestRateLimiter_ResetGlobal(t *testing.T) {
	limiter, mr := setupTestRateLimiter(t, 10)
	defer mr.Close()
	defer limiter.client.Close()

	ctx := context.Background()

	// Reserve for multiple tenants
	for i := 0; i < 3; i++ {
		tenantID := string(rune('a' + i))
		if err := limiter.Reserve(ctx, tenantID); err != nil {
			t.Fatalf("failed to reserve for %s: %v", tenantID, err)
		}
	}

	// Verify global concurrency is 3
	globalConc, err := limiter.GetGlobalConcurrency(ctx)
	if err != nil {
		t.Fatalf("failed to get global concurrency: %v", err)
	}
	if globalConc != 3 {
		t.Fatalf("expected global concurrency 3, got %d", globalConc)
	}

	// Reset global
	if err := limiter.ResetGlobal(ctx); err != nil {
		t.Fatalf("failed to reset global: %v", err)
	}

	// Verify global concurrency is 0
	globalConc, err = limiter.GetGlobalConcurrency(ctx)
	if err != nil {
		t.Fatalf("failed to get global concurrency after reset: %v", err)
	}
	if globalConc != 0 {
		t.Errorf("expected global concurrency 0 after reset, got %d", globalConc)
	}
}
