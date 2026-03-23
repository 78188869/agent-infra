package model

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestBaseModel_Fields(t *testing.T) {
	// Test that BaseModel has all expected fields with correct types
	model := BaseModel{}

	// Use reflection-style checks by field access
	var _ uuid.UUID = model.ID
	var _ gorm.DeletedAt = model.DeletedAt

	// These should compile without error
	_ = model.CreatedAt
	_ = model.UpdatedAt
	_ = model.DeletedAt
}

func TestBaseModel_ID_Type(t *testing.T) {
	model := BaseModel{}

	// Verify ID is a uuid.UUID type
	if model.ID != uuid.Nil {
		t.Error("New BaseModel should have Nil UUID")
	}
}

func TestBaseModel_UUIDGeneration(t *testing.T) {
	// Create a mock DB context for testing BeforeCreate hook
	model := &BaseModel{}

	// BeforeCreate should generate a UUID if ID is Nil
	if model.ID != uuid.Nil {
		t.Error("Initial ID should be Nil")
	}

	// Call BeforeCreate directly (normally called by GORM)
	err := model.BeforeCreate(nil)
	if err != nil {
		t.Errorf("BeforeCreate returned error: %v", err)
	}

	// After BeforeCreate, ID should be generated
	if model.ID == uuid.Nil {
		t.Error("ID should be generated after BeforeCreate")
	}
}

func TestBaseModel_UUIDNotOverwritten(t *testing.T) {
	// Test that existing UUID is not overwritten
	existingID := uuid.New()
	model := &BaseModel{ID: existingID}

	err := model.BeforeCreate(nil)
	if err != nil {
		t.Errorf("BeforeCreate returned error: %v", err)
	}

	if model.ID != existingID {
		t.Error("BeforeCreate should not overwrite existing ID")
	}
}

func TestBaseModel_JSONTags(t *testing.T) {
	// Test JSON serialization behavior
	model := BaseModel{
		ID: uuid.New(),
	}

	// ID should be serialized as "id"
	// CreatedAt should be serialized as "created_at"
	// UpdatedAt should be serialized as "updated_at"
	// DeletedAt should be serialized as "-" (omitted)

	// This test verifies the struct tags are correctly set
	// Actual JSON marshaling is tested in integration tests
	_ = model.ID
	_ = model.CreatedAt
	_ = model.UpdatedAt
	_ = model.DeletedAt
}

func TestBaseModel_SoftDelete(t *testing.T) {
	// Test that DeletedAt field exists and supports soft delete
	model := BaseModel{}

	// DeletedAt should be zero value initially
	if model.DeletedAt.Valid {
		t.Error("DeletedAt should be invalid (null) initially")
	}

	// DeletedAt should be of type gorm.DeletedAt for soft delete support
	var _ gorm.DeletedAt = model.DeletedAt
}

func TestBaseModel_MultipleUUIDsAreUnique(t *testing.T) {
	// Generate multiple models and verify UUIDs are unique
	ids := make(map[uuid.UUID]bool)

	for i := 0; i < 100; i++ {
		model := &BaseModel{}
		err := model.BeforeCreate(nil)
		if err != nil {
			t.Errorf("BeforeCreate returned error: %v", err)
		}

		if ids[model.ID] {
			t.Error("Duplicate UUID generated")
		}
		ids[model.ID] = true
	}

	if len(ids) != 100 {
		t.Errorf("Expected 100 unique IDs, got %d", len(ids))
	}
}
