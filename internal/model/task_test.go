// Package model provides database models for the application.
package model

import (
	"testing"
)

func TestTask_BeforeCreate(t *testing.T) {
	task := &Task{
		TenantID:  "tenant-123",
		CreatorID: "user-123",
		Status:    TaskStatusPending,
		Priority:  PriorityNormal,
	}

	if task.ID != "" {
		t.Error("Task ID should be empty before BeforeCreate")
	}

	err := task.BeforeCreate(nil)
	if err != nil {
		t.Errorf("BeforeCreate() returned error: %v", err)
	}

	if task.ID == "" {
		t.Error("Task ID should be set after BeforeCreate")
	}
}

func TestTask_TableName(t *testing.T) {
	task := Task{}
	if task.TableName() != "tasks" {
		t.Errorf("TableName() = %s, expected tasks", task.TableName())
	}
}

func TestTask_IsTerminal(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected bool
	}{
		{TaskStatusSucceeded, true},
		{TaskStatusFailed, true},
		{TaskStatusCancelled, true},
		{TaskStatusPending, false},
		{TaskStatusRunning, false},
		{TaskStatusPaused, false},
	}

	for _, tt := range tests {
		task := &Task{Status: tt.status}
		if task.IsTerminal() != tt.expected {
			t.Errorf("IsTerminal() for status %s = %v, expected %v", tt.status, task.IsTerminal(), tt.expected)
		}
	}
}

func TestTask_CanPause(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected bool
	}{
		{TaskStatusRunning, true},
		{TaskStatusPending, false},
		{TaskStatusSucceeded, false},
		{TaskStatusPaused, false},
	}

	for _, tt := range tests {
		task := &Task{Status: tt.status}
		if task.CanPause() != tt.expected {
			t.Errorf("CanPause() for status %s = %v, expected %v", tt.status, task.CanPause(), tt.expected)
		}
	}
}

func TestTask_CanResume(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected bool
	}{
		{TaskStatusPaused, true},
		{TaskStatusRunning, false},
		{TaskStatusPending, false},
	}

	for _, tt := range tests {
		task := &Task{Status: tt.status}
		if task.CanResume() != tt.expected {
			t.Errorf("CanResume() for status %s = %v, expected %v", tt.status, task.CanResume(), tt.expected)
		}
	}
}

func TestTask_CanCancel(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected bool
	}{
		{TaskStatusPending, true},
		{TaskStatusScheduled, true},
		{TaskStatusRunning, true},
		{TaskStatusPaused, true},
		{TaskStatusSucceeded, false},
		{TaskStatusFailed, false},
	}

	for _, tt := range tests {
		task := &Task{Status: tt.status}
		if task.CanCancel() != tt.expected {
			t.Errorf("CanCancel() for status %s = %v, expected %v", tt.status, task.CanCancel(), tt.expected)
		}
	}
}

func TestTask_CanInject(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected bool
	}{
		{TaskStatusRunning, true},
		{TaskStatusPending, false},
		{TaskStatusPaused, false},
	}

	for _, tt := range tests {
		task := &Task{Status: tt.status}
		if task.CanInject() != tt.expected {
			t.Errorf("CanInject() for status %s = %v, expected %v", tt.status, task.CanInject(), tt.expected)
		}
	}
}

func TestTask_CanRetry(t *testing.T) {
	// Failed task with retries available
	task := &Task{Status: TaskStatusFailed, RetryCount: 1, MaxRetries: 3}
	if !task.CanRetry() {
		t.Error("Failed task with retries available should be retryable")
	}

	// Failed task with no retries
	task = &Task{Status: TaskStatusFailed, RetryCount: 3, MaxRetries: 3}
	if task.CanRetry() {
		t.Error("Failed task with max retries should not be retryable")
	}

	// Non-failed task
	task = &Task{Status: TaskStatusRunning, RetryCount: 0, MaxRetries: 3}
	if task.CanRetry() {
		t.Error("Running task should not be retryable")
	}
}
