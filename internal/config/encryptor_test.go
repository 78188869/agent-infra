package config

import (
	"strings"
	"testing"
)

func TestAESEncryptor_EncryptDecrypt(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef" // 32 bytes
	enc, err := NewAESEncryptor(key)
	if err != nil {
		t.Fatalf("NewAESEncryptor() error = %v", err)
	}

	plaintext := "my-secret-token-12345"
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if ciphertext == plaintext {
		t.Error("ciphertext should differ from plaintext")
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypt() = %q, want %q", decrypted, plaintext)
	}
}

func TestAESEncryptor_DifferentCiphertexts(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef"
	enc, err := NewAESEncryptor(key)
	if err != nil {
		t.Fatalf("NewAESEncryptor() error = %v", err)
	}

	plaintext := "same-input"
	ct1, _ := enc.Encrypt(plaintext)
	ct2, _ := enc.Encrypt(plaintext)

	if ct1 == ct2 {
		t.Error("two encryptions of same plaintext should produce different ciphertexts (random nonce)")
	}
}

func TestAESEncryptor_InvalidKeyLength(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"too short", "short"},
		{"too long", strings.Repeat("a", 64)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAESEncryptor(tt.key)
			if err == nil {
				t.Errorf("NewAESEncryptor(%q) expected error, got nil", tt.name)
			}
		})
	}
}

func TestAESEncryptor_InvalidCiphertext(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef"
	enc, err := NewAESEncryptor(key)
	if err != nil {
		t.Fatalf("NewAESEncryptor() error = %v", err)
	}

	tests := []struct {
		name string
		ct   string
	}{
		{"not base64", "!!!invalid!!!"},
		{"too short", "aa"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := enc.Decrypt(tt.ct)
			if err == nil {
				t.Errorf("Decrypt(%q) expected error, got nil", tt.name)
			}
		})
	}
}

func TestAESEncryptor_EmptyPlaintext(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef"
	enc, err := NewAESEncryptor(key)
	if err != nil {
		t.Fatalf("NewAESEncryptor() error = %v", err)
	}

	ct, err := enc.Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt('') error = %v", err)
	}

	pt, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if pt != "" {
		t.Errorf("Decrypt() = %q, want empty string", pt)
	}
}
