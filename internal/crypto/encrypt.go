package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Encryptor provides encryption and decryption for sensitive data.
// Used to encrypt provider API keys and credentials before storing in database.
// Uses AES-256-GCM for authenticated encryption.
type Encryptor interface {
	// Encrypt encrypts plaintext and returns base64-encoded ciphertext.
	// Uses AES-256-GCM for authenticated encryption with random nonce.
	Encrypt(plaintext []byte) ([]byte, error)

	// Decrypt decrypts base64-encoded ciphertext and returns plaintext.
	// Verifies authentication tag to detect tampering.
	Decrypt(ciphertext []byte) ([]byte, error)
}

// AESEncryptor implements Encryptor using AES-256-GCM.
// The encryption key should be 32 bytes (256 bits) and stored securely
// (environment variable, secrets manager, etc.).
type AESEncryptor struct {
	key []byte // 32-byte encryption key
}

// NewAESEncryptor creates an AES-256-GCM encryptor.
// The key must be exactly 32 bytes for AES-256.
// Key should be generated using a cryptographically secure random number generator
// and stored in environment variable or secrets manager.
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	if key == nil {
		return nil, fmt.Errorf("encryption key cannot be nil")
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes for AES-256")
	}
	return &AESEncryptor{key: key}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns base64-encoded ciphertext.
// The output format is: base64(nonce + ciphertext + tag)
// where nonce is 12 bytes and tag is 16 bytes.
func (e *AESEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return []byte(encoded), nil
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM and returns plaintext.
// Expects input format: base64(nonce + ciphertext + tag)
func (e *AESEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(ciphertext))
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(decoded) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := decoded[:gcm.NonceSize()]
	ciphertextWithTag := decoded[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertextWithTag, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// GenerateKey generates a cryptographically secure 32-byte key for AES-256.
// Use this to generate a new encryption key, then store it securely.
// This function is provided for convenience during setup/testing.
// In production, use a proper key management system.
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// EncodeKeyBase64 encodes an encryption key as base64 for storage in env vars.
func EncodeKeyBase64(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// DecodeKeyBase64 decodes a base64-encoded encryption key from env vars.
func DecodeKeyBase64(encodedKey string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key length after base64 decode")
	}
	return key, nil
}
