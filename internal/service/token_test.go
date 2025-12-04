package service

import (
	"testing"
)

func TestGenerateToken(t *testing.T) {
	// Test that tokens are generated with correct length
	token, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken() error = %v", err)
	}

	// Token should be hex-encoded, so 32 bytes = 64 hex chars
	expectedLen := TokenLength * 2
	if len(token) != expectedLen {
		t.Errorf("generateToken() length = %d, want %d", len(token), expectedLen)
	}

	// Test that tokens are unique
	token2, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken() second call error = %v", err)
	}

	if token == token2 {
		t.Error("generateToken() produced duplicate tokens")
	}
}

func TestHashToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "simple token",
			token: "abc123",
		},
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "long token",
			token: "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashToken(tt.token)

			// SHA-256 produces 32 bytes = 64 hex chars
			if len(hash) != 64 {
				t.Errorf("hashToken() length = %d, want 64", len(hash))
			}

			// Same input should produce same output
			hash2 := hashToken(tt.token)
			if hash != hash2 {
				t.Error("hashToken() is not deterministic")
			}
		})
	}

	// Different inputs should produce different outputs
	hash1 := hashToken("token1")
	hash2 := hashToken("token2")
	if hash1 == hash2 {
		t.Error("hashToken() produced same hash for different inputs")
	}
}

func TestGenerateVerificationToken(t *testing.T) {
	token, err := generateVerificationToken()
	if err != nil {
		t.Fatalf("generateVerificationToken() error = %v", err)
	}

	// Token should be hex-encoded, so 32 bytes = 64 hex chars
	expectedLen := VerificationTokenLength * 2
	if len(token) != expectedLen {
		t.Errorf("generateVerificationToken() length = %d, want %d", len(token), expectedLen)
	}

	// Test uniqueness
	token2, err := generateVerificationToken()
	if err != nil {
		t.Fatalf("generateVerificationToken() second call error = %v", err)
	}

	if token == token2 {
		t.Error("generateVerificationToken() produced duplicate tokens")
	}
}

func TestHashVerificationToken(t *testing.T) {
	token := "test-verification-token"
	hash := hashVerificationToken(token)

	// SHA-256 produces 64 hex chars
	if len(hash) != 64 {
		t.Errorf("hashVerificationToken() length = %d, want 64", len(hash))
	}

	// Deterministic
	hash2 := hashVerificationToken(token)
	if hash != hash2 {
		t.Error("hashVerificationToken() is not deterministic")
	}

	// Different inputs produce different outputs
	differentHash := hashVerificationToken("different-token")
	if hash == differentHash {
		t.Error("hashVerificationToken() produced same hash for different inputs")
	}
}
