package model

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func TestIntervention_Fields(t *testing.T) {
	// Test that Intervention has all expected fields with correct types
	intervention := Intervention{}

	// Verify all fields exist and are accessible
	_ = intervention.ID
	_ = intervention.CreatedAt
	_ = intervention.UpdatedAt
	_ = intervention.DeletedAt
	_ = intervention.TaskID
	_ = intervention.OperatorID
	_ = intervention.Action
	_ = intervention.Content
	_ = intervention.Reason
	_ = intervention.Result
	_ = intervention.Status
}

func TestIntervention_EmbedsBaseModel(t *testing.T) {
	// Test that Intervention embeds BaseModel
	intervention := Intervention{}

	// ID should be uuid.UUID type (from BaseModel)
	var _ uuid.UUID = intervention.ID

	// DeletedAt should support soft delete (from BaseModel)
	_ = intervention.DeletedAt
}

func TestIntervention_DefaultValues(t *testing.T) {
	// Test default values for a new intervention
	intervention := Intervention{}

	// Zero values should be the default
	if intervention.TaskID != "" {
		t.Error("Default TaskID should be empty string")
	}
	if intervention.OperatorID != "" {
		t.Error("Default OperatorID should be empty string")
	}
	if intervention.Action != "" {
		t.Error("Default Action should be empty string")
	}
	if intervention.Reason != "" {
		t.Error("Default Reason should be empty string")
	}
	if intervention.Status != "" {
		t.Error("Default Status should be empty string")
	}
}

func TestIntervention_ActionConstants(t *testing.T) {
	// Test that action constants are defined
	tests := []struct {
		name     string
		constant InterventionAction
		expected string
	}{
		{"Pause", InterventionActionPause, "pause"},
		{"Resume", InterventionActionResume, "resume"},
		{"Cancel", InterventionActionCancel, "cancel"},
		{"Inject", InterventionActionInject, "inject"},
		{"Modify", InterventionActionModify, "modify"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("%s should be '%s', got '%s'", tt.name, tt.expected, tt.constant)
			}
		})
	}
}

func TestIntervention_StatusConstants(t *testing.T) {
	// Test that status constants are defined
	tests := []struct {
		name     string
		constant InterventionStatus
		expected string
	}{
		{"Pending", InterventionStatusPending, "pending"},
		{"Applied", InterventionStatusApplied, "applied"},
		{"Failed", InterventionStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("%s should be '%s', got '%s'", tt.name, tt.expected, tt.constant)
			}
		})
	}
}

func TestIntervention_TableName(t *testing.T) {
	// Test that TableName returns the correct table name
	intervention := Intervention{}
	if intervention.TableName() != "interventions" {
		t.Errorf("TableName should return 'interventions', got '%s'", intervention.TableName())
	}
}

func TestIntervention_JSONFields(t *testing.T) {
	// Test that JSON fields can be set and read
	intervention := Intervention{
		TaskID:     "task-uuid-123",
		OperatorID: "operator-uuid-456",
		Action:     InterventionActionPause,
		Content:    datatypes.JSON(`{"instruction": "test"}`),
		Reason:     "Test intervention",
		Result:     datatypes.JSON(`{"success": true}`),
		Status:     InterventionStatusPending,
	}

	if intervention.TaskID != "task-uuid-123" {
		t.Errorf("Expected TaskID 'task-uuid-123', got '%s'", intervention.TaskID)
	}
	if intervention.OperatorID != "operator-uuid-456" {
		t.Errorf("Expected OperatorID 'operator-uuid-456', got '%s'", intervention.OperatorID)
	}
	if intervention.Action != InterventionActionPause {
		t.Errorf("Expected Action '%s', got '%s'", InterventionActionPause, intervention.Action)
	}
	if intervention.Reason != "Test intervention" {
		t.Errorf("Expected Reason 'Test intervention', got '%s'", intervention.Reason)
	}
	if intervention.Status != InterventionStatusPending {
		t.Errorf("Expected Status '%s', got '%s'", InterventionStatusPending, intervention.Status)
	}
}

func TestIntervention_BeforeCreate(t *testing.T) {
	// Test that BeforeCreate (from BaseModel) generates a UUID if ID is empty
	intervention := &Intervention{}

	if intervention.ID != uuid.Nil {
		t.Error("Initial ID should be Nil")
	}

	// Call BeforeCreate directly (inherited from BaseModel)
	err := intervention.BaseModel.BeforeCreate(nil)
	if err != nil {
		t.Errorf("BeforeCreate returned error: %v", err)
	}

	// After BeforeCreate, ID should be generated
	if intervention.ID == uuid.Nil {
		t.Error("ID should be generated after BeforeCreate")
	}
}

func TestIntervention_BeforeCreateNotOverwritten(t *testing.T) {
	// Test that existing UUID is not overwritten (inherited from BaseModel)
	existingID := uuid.New()
	intervention := &Intervention{}
	intervention.ID = existingID

	err := intervention.BaseModel.BeforeCreate(nil)
	if err != nil {
		t.Errorf("BeforeCreate returned error: %v", err)
	}

	if intervention.ID != existingID {
		t.Error("BeforeCreate should not overwrite existing ID")
	}
}

func TestIntervention_StatusHelpers(t *testing.T) {
	// Test IsPending, IsApplied, IsFailed helper methods
	tests := []struct {
		name           string
		status         InterventionStatus
		isPending      bool
		isApplied      bool
		isFailed       bool
	}{
		{"Pending", InterventionStatusPending, true, false, false},
		{"Applied", InterventionStatusApplied, false, true, false},
		{"Failed", InterventionStatusFailed, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intervention := &Intervention{Status: tt.status}
			if intervention.IsPending() != tt.isPending {
				t.Errorf("IsPending() should be %v for status %s", tt.isPending, tt.status)
			}
			if intervention.IsApplied() != tt.isApplied {
				t.Errorf("IsApplied() should be %v for status %s", tt.isApplied, tt.status)
			}
			if intervention.IsFailed() != tt.isFailed {
				t.Errorf("IsFailed() should be %v for status %s", tt.isFailed, tt.status)
			}
		})
	}
}

func TestIntervention_Relations(t *testing.T) {
	// Test that relation fields exist
	intervention := Intervention{}

	// Relation fields should exist
	_ = intervention.Task
	_ = intervention.Operator
}
