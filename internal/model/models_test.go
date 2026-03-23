// Package model provides database models for the application.
package model

import (
	"testing"
)

func TestGenerateUUID(t *testing.T) {
	uuid := generateUUID()

	if uuid == "" {
		t.Error("generateUUID() returned empty string")
	}

	if len(uuid) != 36 {
		t.Errorf("generateUUID() returned string of length %d, expected 36", len(uuid))
	}
}

func TestAllModels(t *testing.T) {
	models := AllModels()

	expectedModels := 10
	if len(models) != expectedModels {
		t.Errorf("AllModels() returned %d models, expected %d", len(models), expectedModels)
	}

	// Verify each model is non-nil
	for i, mdl := range models {
		if mdl == nil {
			t.Errorf("AllModels()[%d] is nil", i)
		}
	}
}
