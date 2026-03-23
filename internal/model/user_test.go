// Package model provides database models for the application.
package model

import (
	"testing"
)

func TestUser_BeforeCreate(t *testing.T) {
	user := &User{
		TenantID: "tenant-123",
		Username: "testuser",
		Role:     UserRoleDeveloper,
		Status:   UserStatusActive,
	}

	if user.ID != "" {
		t.Error("User ID should be empty before BeforeCreate")
	}

	err := user.BeforeCreate(nil)
	if err != nil {
		t.Errorf("BeforeCreate() returned error: %v", err)
	}

	if user.ID == "" {
		t.Error("User ID should be set after BeforeCreate")
	}
}

func TestUser_TableName(t *testing.T) {
	user := User{}
	if user.TableName() != "users" {
		t.Errorf("TableName() = %s, expected users", user.TableName())
	}
}

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		role     UserRole
		expected bool
	}{
		{UserRoleAdmin, true},
		{UserRoleDeveloper, false},
		{UserRoleOperator, false},
		{UserRoleReviewer, false},
	}

	for _, tt := range tests {
		user := &User{Role: tt.role}
		if user.IsAdmin() != tt.expected {
			t.Errorf("IsAdmin() for role %s = %v, expected %v", tt.role, user.IsAdmin(), tt.expected)
		}
	}
}

func TestUser_IsActive(t *testing.T) {
	tests := []struct {
		status   UserStatus
		expected bool
	}{
		{UserStatusActive, true},
		{UserStatusDisabled, false},
	}

	for _, tt := range tests {
		user := &User{Status: tt.status}
		if user.IsActive() != tt.expected {
			t.Errorf("IsActive() for status %s = %v, expected %v", tt.status, user.IsActive(), tt.expected)
		}
	}
}
