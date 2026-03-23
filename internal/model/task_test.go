package model

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func TestTask_Fields(t *testing.T) {
	// Test that Task has all expected fields with correct types
	task := Task{}

	// Verify all fields exist and are accessible
	_ = task.ID
	_ = task.CreatedAt
	_ = task.UpdatedAt
	_ = task.DeletedAt
	_ = task.TenantID
	_ = task.TemplateID
	_ = task.CreatorID
	_ = task.ProviderID
	_ = task.Name
	_ = task.Status
	_ = task.Priority
	_ = task.Params
	_ = task.Description
	_ = task.ErrorMessage
	_ = task.Result
}

func TestTask_EmbedsBaseModel(t *testing.T) {
	// Test that Task embeds BaseModel
	task := Task{}

	// ID should be uuid.UUID type (from BaseModel)
	var _ uuid.UUID = task.ID

	// DeletedAt should support soft delete (from BaseModel)
	_ = task.DeletedAt
}

func TestTask_DefaultValues(t *testing.T) {
	// Test default values for a new task
	task := Task{}

	// Zero values should be the default
	if task.TenantID != "" {
		t.Error("Default TenantID should be empty string")
	}
	if task.TemplateID != nil {
		t.Error("Default TemplateID should be nil")
	}
	if task.CreatorID != "" {
		t.Error("Default CreatorID should be empty string")
	}
	if task.ProviderID != "" {
		t.Error("Default ProviderID should be empty string")
	}
	if task.Name != "" {
		t.Error("Default Name should be empty string")
	}
	if task.Status != "" {
		t.Error("Default Status should be empty string")
	}
	if task.Priority != "" {
		t.Error("Default Priority should be empty string")
	}
	if task.Description != "" {
		t.Error("Default Description should be empty string")
	}
	if task.ErrorMessage != "" {
		t.Error("Default ErrorMessage should be empty string")
	}
}

func TestTask_StatusConstants(t *testing.T) {
	// Test that status constants are defined
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"Pending", TaskStatusPending, "pending"},
		{"Scheduled", TaskStatusScheduled, "scheduled"},
		{"Running", TaskStatusRunning, "running"},
		{"Paused", TaskStatusPaused, "paused"},
		{"WaitingApproval", TaskStatusWaitingApproval, "waiting_approval"},
		{"Retrying", TaskStatusRetrying, "retrying"},
		{"Succeeded", TaskStatusSucceeded, "succeeded"},
		{"Failed", TaskStatusFailed, "failed"},
		{"Cancelled", TaskStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s should be '%s', got '%s'", tt.name, tt.expected, tt.constant)
			}
		})
	}
}

func TestTask_PriorityConstants(t *testing.T) {
	// Test that priority constants are defined
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"High", TaskPriorityHigh, "high"},
		{"Normal", TaskPriorityNormal, "normal"},
		{"Low", TaskPriorityLow, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s should be '%s', got '%s'", tt.name, tt.expected, tt.constant)
			}
		})
	}
}

func TestTask_TableName(t *testing.T) {
	// Test that TableName returns the correct table name
	task := Task{}
	if task.TableName() != "tasks" {
		t.Errorf("TableName should return 'tasks', got '%s'", task.TableName())
	}
}

func TestTask_JSONFields(t *testing.T) {
	// Test that JSON fields can be set and read
	task := Task{
		Name:     "Test Task",
		Status:   TaskStatusPending,
		Priority: TaskPriorityHigh,
		Params:   datatypes.JSON(`{"key": "value"}`),
		Result:   datatypes.JSON(`{"output": "success"}`),
	}

	if task.Name != "Test Task" {
		t.Errorf("Expected Name 'Test Task', got '%s'", task.Name)
	}
	if task.Status != TaskStatusPending {
		t.Errorf("Expected Status '%s', got '%s'", TaskStatusPending, task.Status)
	}
	if task.Priority != TaskPriorityHigh {
		t.Errorf("Expected Priority '%s', got '%s'", TaskPriorityHigh, task.Priority)
	}
}

func TestTask_TemplateIDNil(t *testing.T) {
	// Test that TemplateID can be nil (optional field)
	task := Task{}

	if task.TemplateID != nil {
		t.Error("TemplateID should be nil by default")
	}

	// Test setting TemplateID
	templateID := "template-123"
	task.TemplateID = &templateID
	if task.TemplateID == nil || *task.TemplateID != "template-123" {
		t.Error("TemplateID should be settable")
	}
}

func TestTask_PointerFields(t *testing.T) {
	// Test pointer fields for nullable relationships
	templateID := "template-uuid-123"
	task := Task{
		TenantID:   "tenant-uuid",
		TemplateID: &templateID,
		CreatorID:  "creator-uuid",
		ProviderID: "provider-uuid",
	}

	if task.TemplateID == nil {
		t.Error("TemplateID should not be nil")
	}
	if *task.TemplateID != "template-uuid-123" {
		t.Errorf("Expected TemplateID 'template-uuid-123', got '%s'", *task.TemplateID)
	}
}
