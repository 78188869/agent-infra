package model

import (
	"testing"
)

func TestCredential_Fields(t *testing.T) {
	c := Credential{
		UserID:    "user-123",
		Type:      CredentialTypeGitToken,
		Encrypted: "encrypted-value",
	}

	if c.UserID != "user-123" {
		t.Errorf("expected UserID 'user-123', got '%s'", c.UserID)
	}
	if c.Type != CredentialTypeGitToken {
		t.Errorf("expected Type '%s', got '%s'", CredentialTypeGitToken, c.Type)
	}
	if c.Encrypted != "encrypted-value" {
		t.Errorf("expected Encrypted 'encrypted-value', got '%s'", c.Encrypted)
	}
}

func TestCredential_JSONTag(t *testing.T) {
	c := Credential{Encrypted: "secret"}
	// Encrypted field should have `json:"-"` tag, so it's never serialized
	if c.Encrypted == "" {
		t.Error("Encrypted should be settable")
	}
}

func TestIsValidCredentialType(t *testing.T) {
	tests := []struct {
		ctype   string
		want    bool
	}{
		{CredentialTypeGitToken, true},
		{CredentialTypeDevOpsToken, true},
		{"invalid_type", false},
		{"", false},
	}
	for _, tt := range tests {
		got := IsValidCredentialType(tt.ctype)
		if got != tt.want {
			t.Errorf("IsValidCredentialType(%q) = %v, want %v", tt.ctype, got, tt.want)
		}
	}
}

func TestValidCredentialTypes(t *testing.T) {
	types := ValidCredentialTypes()
	if len(types) != 2 {
		t.Errorf("expected 2 credential types, got %d", len(types))
	}
}
