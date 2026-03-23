package model

import (
	"testing"

	"github.com/google/uuid"
)

func TestTenant_Fields(t *testing.T) {
	// Test that Tenant has all expected fields with correct types
	tenant := Tenant{}

	// Verify all fields exist and are accessible
	_ = tenant.ID
	_ = tenant.CreatedAt
	_ = tenant.UpdatedAt
	_ = tenant.DeletedAt
	_ = tenant.Name
	_ = tenant.QuotaCPU
	_ = tenant.QuotaMemory
	_ = tenant.QuotaConcurrency
	_ = tenant.QuotaDailyTasks
	_ = tenant.Status
}

func TestTenant_EmbedsBaseModel(t *testing.T) {
	// Test that Tenant embeds BaseModel
	tenant := Tenant{}

	// ID should be uuid.UUID type (from BaseModel)
	var _ uuid.UUID = tenant.ID

	// DeletedAt should support soft delete (from BaseModel)
	_ = tenant.DeletedAt
}

func TestTenant_DefaultValues(t *testing.T) {
	// Test default values for a new tenant
	tenant := Tenant{}

	// Zero values should be the default
	if tenant.Name != "" {
		t.Error("Default Name should be empty string")
	}
	if tenant.QuotaCPU != 0 {
		t.Error("Default QuotaCPU should be 0")
	}
	if tenant.QuotaMemory != 0 {
		t.Error("Default QuotaMemory should be 0")
	}
	if tenant.QuotaConcurrency != 0 {
		t.Error("Default QuotaConcurrency should be 0")
	}
	if tenant.QuotaDailyTasks != 0 {
		t.Error("Default QuotaDailyTasks should be 0")
	}
	if tenant.Status != "" {
		t.Error("Default Status should be empty string")
	}
}

func TestTenant_StatusConstants(t *testing.T) {
	// Test that status constants are defined
	if TenantStatusActive != "active" {
		t.Errorf("TenantStatusActive should be 'active', got '%s'", TenantStatusActive)
	}
	if TenantStatusSuspended != "suspended" {
		t.Errorf("TenantStatusSuspended should be 'suspended', got '%s'", TenantStatusSuspended)
	}
}

func TestTenant_TableName(t *testing.T) {
	// Test that TableName returns the correct table name
	tenant := Tenant{}
	if tenant.TableName() != "tenants" {
		t.Errorf("TableName should return 'tenants', got '%s'", tenant.TableName())
	}
}
