// Package model provides database models for the application.
package model

import (
	"testing"
	"time"
)

func TestAPIKey_BeforeCreate(t *testing.T) {
	key := &APIKey{
		UserID:    "user-123",
		KeyHash:   "hash123",
		KeyPrefix: "sk_test",
		Status:    APIKeyStatusActive,
	}

	if key.ID != "" {
		t.Error("APIKey ID should be empty before BeforeCreate")
	}

	err := key.BeforeCreate(nil)
	if err != nil {
		t.Errorf("BeforeCreate() returned error: %v", err)
	}

	if key.ID == "" {
		t.Error("APIKey ID should be set after BeforeCreate")
	}
}

func TestAPIKey_IsActive(t *testing.T) {
	// Active key
	key := &APIKey{Status: APIKeyStatusActive}
	if !key.IsActive() {
		t.Error("Active key should be active")
	}

	// Revoked key
	key = &APIKey{Status: APIKeyStatusRevoked}
	if key.IsActive() {
		t.Error("Revoked key should not be active")
	}

	// Expired key
	pastTime := time.Now().Add(-24 * time.Hour)
	key = &APIKey{Status: APIKeyStatusActive, ExpiresAt: &pastTime}
	if key.IsActive() {
		t.Error("Expired key should not be active")
	}
}

func TestAPIKey_IsExpired(t *testing.T) {
	// No expiration
	key := &APIKey{}
	if key.IsExpired() {
		t.Error("Key without expiration should not be expired")
	}

	// Past expiration
	pastTime := time.Now().Add(-24 * time.Hour)
	key = &APIKey{ExpiresAt: &pastTime}
	if !key.IsExpired() {
		t.Error("Key with past expiration should be expired")
	}

	// Future expiration
	futureTime := time.Now().Add(24 * time.Hour)
	key = &APIKey{ExpiresAt: &futureTime}
	if key.IsExpired() {
		t.Error("Key with future expiration should not be expired")
	}
}

func TestHashKey(t *testing.T) {
	key := "test-api-key-123"
	hash := HashKey(key)

	if hash == "" {
		t.Error("HashKey() returned empty string")
	}

	if len(hash) != 64 {
		t.Errorf("HashKey() returned hash of length %d, expected 64", len(hash))
	}

	// Same key should produce same hash
	hash2 := HashKey(key)
	if hash != hash2 {
		t.Error("HashKey() should produce consistent hash")
	}
}

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"sk_test_1234567890", "sk_test_"},
		{"short", "short"},
		{"12345678", "12345678"},
	}

	for _, tt := range tests {
		result := ExtractPrefix(tt.key)
		if result != tt.expected {
			t.Errorf("ExtractPrefix(%s) = %s, expected %s", tt.key, result, tt.expected)
		}
	}
}
