// Package model provides database models for the application.
package model

import (
	"testing"
)

func TestTenant_BeforeCreate(t *testing.T) {
	tenant := &Tenant{
		Name:   "test-tenant",
		Status: TenantStatusActive,
	}

	if tenant.ID != "" {
		t.Error("Tenant ID should be empty before BeforeCreate")
	}

	// Simulate BeforeCreate
	err := tenant.BeforeCreate(nil)
	if err != nil {
		t.Errorf("BeforeCreate() returned error: %v", err)
	}

	if tenant.ID == "" {
		t.Error("Tenant ID should be set after BeforeCreate")
	}
}

func TestTenant_TableName(t *testing.T) {
	tenant := Tenant{}
	if tenant.TableName() != "tenants" {
		t.Errorf("TableName() = %s, expected tenants", tenant.TableName())
	}
}

func TestTenantStatus(t *testing.T) {
	tests := []struct {
		status   TenantStatus
		expected string
	}{
		{TenantStatusActive, "active"},
		{TenantStatusSuspended, "suspended"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("TenantStatus = %s, expected %s", tt.status, tt.expected)
		}
	}
}
