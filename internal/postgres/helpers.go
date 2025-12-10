package postgres

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// generateSessionID generates a cryptographically secure session ID.
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
