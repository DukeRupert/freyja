package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLength is the minimum acceptable password length
	MinPasswordLength = 8

	// bcryptCost is the cost factor for bcrypt hashing
	// 12 is a good balance between security and performance (2025)
	bcryptCost = 12
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordMismatch = errors.New("password does not match")
)

// HashPassword generates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength {
		return "", ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword checks if the provided password matches the hash
func VerifyPassword(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrPasswordMismatch
		}
		return fmt.Errorf("failed to verify password: %w", err)
	}
	return nil
}
