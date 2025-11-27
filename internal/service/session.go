package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateSessionID generates a cryptographically secure session ID
// Uses 32 bytes of random data encoded as base64 URL-safe string
func GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return base64.URLEncoding.EncodeToString(b), nil
}
