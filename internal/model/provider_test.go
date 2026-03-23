// Package model provides database models for the application.
package model

import (
	"testing"
)

func TestProvider_BeforeCreate(t *testing.T) {
	provider := &Provider{
		Scope:   ProviderScopeSystem,
		Name:    "test-provider",
		Type:    ProviderTypeClaudeCode,
		Status:  ProviderStatusActive,
	}

	if provider.ID != "" {
		t.Error("Provider ID should be empty before BeforeCreate")
	}

	err := provider.BeforeCreate(nil)
	if err != nil {
		t.Errorf("BeforeCreate() returned error: %v", err)
	}

	if provider.ID == "" {
		t.Error("Provider ID should be set after BeforeCreate")
	}
}

func TestProvider_TableName(t *testing.T) {
	provider := Provider{}
	if provider.TableName() != "providers" {
		t.Errorf("TableName() = %s, expected providers", provider.TableName())
	}
}

func TestProvider_IsActive(t *testing.T) {
	tests := []struct {
		status   ProviderStatus
		expected bool
	}{
		{ProviderStatusActive, true},
		{ProviderStatusInactive, false},
		{ProviderStatusDeprecated, false},
	}

	for _, tt := range tests {
		provider := &Provider{Status: tt.status}
		if provider.IsActive() != tt.expected {
			t.Errorf("IsActive() for status %s = %v, expected %v", tt.status, provider.IsActive(), tt.expected)
		}
	}
}

func TestProvider_ScopeChecks(t *testing.T) {
	// System provider
	provider := &Provider{Scope: ProviderScopeSystem}
	if !provider.IsSystemProvider() {
		t.Error("System scope provider should be system provider")
	}
	if provider.IsTenantProvider() {
		t.Error("System scope provider should not be tenant provider")
	}
	if provider.IsUserProvider() {
		t.Error("System scope provider should not be user provider")
	}

	// Tenant provider
	tenantID := "tenant-123"
	provider = &Provider{Scope: ProviderScopeTenant, TenantID: &tenantID}
	if provider.IsSystemProvider() {
		t.Error("Tenant scope provider should not be system provider")
	}
	if !provider.IsTenantProvider() {
		t.Error("Tenant scope provider should be tenant provider")
	}
	if provider.IsUserProvider() {
		t.Error("Tenant scope provider should not be user provider")
	}

	// User provider
	userID := "user-123"
	provider = &Provider{Scope: ProviderScopeUser, TenantID: &tenantID, UserID: &userID}
	if provider.IsSystemProvider() {
		t.Error("User scope provider should not be system provider")
	}
	if provider.IsTenantProvider() {
		t.Error("User scope provider should not be tenant provider")
	}
	if !provider.IsUserProvider() {
		t.Error("User scope provider should be user provider")
	}
}

func TestProviderType(t *testing.T) {
	types := []ProviderType{
		ProviderTypeClaudeCode,
		ProviderTypeAnthropicCompat,
		ProviderTypeOpenAICompat,
		ProviderTypeCustom,
	}

	for _, pt := range types {
		if string(pt) == "" {
			t.Errorf("ProviderType %s has empty string representation", pt)
		}
	}
}
