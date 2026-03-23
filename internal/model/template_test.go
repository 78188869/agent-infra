package model

import (
	"testing"

	"github.com/google/uuid"
)

func TestTemplate_Fields(t *testing.T) {
	// Test that Template has all expected fields with correct types
	template := Template{}

	// Verify all fields exist and are accessible
	_ = template.ID
	_ = template.CreatedAt
	_ = template.UpdatedAt
	_ = template.DeletedAt
	_ = template.TenantID
	_ = template.Name
	_ = template.Version
	_ = template.Spec
	_ = template.SceneType
	_ = template.Status
	_ = template.ProviderID
}

func TestTemplate_EmbedsBaseModel(t *testing.T) {
	// Test that Template embeds BaseModel
	template := Template{}

	// ID should be uuid.UUID type (from BaseModel)
	var _ uuid.UUID = template.ID

	// DeletedAt should support soft delete (from BaseModel)
	_ = template.DeletedAt
}

func TestTemplate_DefaultValues(t *testing.T) {
	// Test default values for a new template
	template := Template{}

	// Zero values should be the default
	if template.Name != "" {
		t.Error("Default Name should be empty string")
	}
	if template.Version != "" {
		t.Error("Default Version should be empty string")
	}
	if template.Spec != "" {
		t.Error("Default Spec should be empty string")
	}
	if template.SceneType != "" {
		t.Error("Default SceneType should be empty string")
	}
	if template.Status != "" {
		t.Error("Default Status should be empty string")
	}
	if template.ProviderID != nil {
		t.Error("Default ProviderID should be nil")
	}
}

func TestTemplate_SceneTypeConstants(t *testing.T) {
	// Test that scene type constants are defined
	if TemplateSceneTypeCoding != "coding" {
		t.Errorf("TemplateSceneTypeCoding should be 'coding', got '%s'", TemplateSceneTypeCoding)
	}
	if TemplateSceneTypeOps != "ops" {
		t.Errorf("TemplateSceneTypeOps should be 'ops', got '%s'", TemplateSceneTypeOps)
	}
	if TemplateSceneTypeAnalysis != "analysis" {
		t.Errorf("TemplateSceneTypeAnalysis should be 'analysis', got '%s'", TemplateSceneTypeAnalysis)
	}
	if TemplateSceneTypeContent != "content" {
		t.Errorf("TemplateSceneTypeContent should be 'content', got '%s'", TemplateSceneTypeContent)
	}
	if TemplateSceneTypeCustom != "custom" {
		t.Errorf("TemplateSceneTypeCustom should be 'custom', got '%s'", TemplateSceneTypeCustom)
	}
}

func TestTemplate_StatusConstants(t *testing.T) {
	// Test that status constants are defined
	if TemplateStatusDraft != "draft" {
		t.Errorf("TemplateStatusDraft should be 'draft', got '%s'", TemplateStatusDraft)
	}
	if TemplateStatusPublished != "published" {
		t.Errorf("TemplateStatusPublished should be 'published', got '%s'", TemplateStatusPublished)
	}
	if TemplateStatusDeprecated != "deprecated" {
		t.Errorf("TemplateStatusDeprecated should be 'deprecated', got '%s'", TemplateStatusDeprecated)
	}
}

func TestTemplate_TableName(t *testing.T) {
	// Test that TableName returns the correct table name
	template := Template{}
	if template.TableName() != "templates" {
		t.Errorf("TableName should return 'templates', got '%s'", template.TableName())
	}
}

func TestTemplate_ProviderIDNilable(t *testing.T) {
	// Test that ProviderID can be nil or set to a value
	template := Template{}

	// Should be nil by default
	if template.ProviderID != nil {
		t.Error("ProviderID should be nil by default")
	}

	// Should be settable
	id := uuid.New().String()
	template.ProviderID = &id
	if template.ProviderID == nil {
		t.Error("ProviderID should not be nil after assignment")
	}
	if *template.ProviderID != id {
		t.Errorf("ProviderID should be '%s', got '%s'", id, *template.ProviderID)
	}
}
