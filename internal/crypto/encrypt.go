package crypto

import (
	"fmt"

	// These imports are used in the implementation TODOs
	_ "crypto/aes"
	_ "crypto/cipher"
	_ "crypto/rand"
	_ "encoding/base64"
	_ "io"
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
	// TODO: Validate that key is not nil
	// TODO: Validate that key is exactly 32 bytes (256 bits)
	// TODO: If key length is wrong, return error: "encryption key must be 32 bytes for AES-256"
	// TODO: Create AESEncryptor with key
	// TODO: Return initialized encryptor
	return nil, fmt.Errorf("not implemented")
}

// Encrypt encrypts plaintext using AES-256-GCM and returns base64-encoded ciphertext.
// The output format is: base64(nonce + ciphertext + tag)
// where nonce is 12 bytes and tag is 16 bytes.
func (e *AESEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	// TODO: Create AES cipher block using aes.NewCipher(e.key)
	// TODO: Handle cipher creation error
	// TODO: Create GCM mode using cipher.NewGCM(block)
	// TODO: Handle GCM creation error
	// TODO: Generate random nonce: make([]byte, gcm.NonceSize())
	// TODO: Fill nonce with crypto random bytes using io.ReadFull(rand.Reader, nonce)
	// TODO: Handle random generation error
	// TODO: Encrypt plaintext: gcm.Seal(nonce, nonce, plaintext, nil)
	//       Note: Seal appends ciphertext+tag to nonce, so result is nonce+ciphertext+tag
	// TODO: Encode result as base64: base64.StdEncoding.EncodeToString(ciphertext)
	// TODO: Return base64-encoded ciphertext as []byte
	return nil, fmt.Errorf("not implemented")
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM and returns plaintext.
// Expects input format: base64(nonce + ciphertext + tag)
func (e *AESEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	// TODO: Decode base64 ciphertext: base64.StdEncoding.DecodeString(string(ciphertext))
	// TODO: Handle base64 decode error
	// TODO: Create AES cipher block using aes.NewCipher(e.key)
	// TODO: Handle cipher creation error
	// TODO: Create GCM mode using cipher.NewGCM(block)
	// TODO: Handle GCM creation error
	// TODO: Validate that decoded data is at least nonce size (gcm.NonceSize())
	// TODO: If too short, return error: "ciphertext too short"
	// TODO: Split decoded data into nonce and ciphertext+tag:
	//       nonce := decoded[:gcm.NonceSize()]
	//       ciphertextWithTag := decoded[gcm.NonceSize():]
	// TODO: Decrypt and verify: gcm.Open(nil, nonce, ciphertextWithTag, nil)
	// TODO: Handle decryption/verification error (could be wrong key or tampered data)
	// TODO: Return plaintext
	return nil, fmt.Errorf("not implemented")
}

// GenerateKey generates a cryptographically secure 32-byte key for AES-256.
// Use this to generate a new encryption key, then store it securely.
// This function is provided for convenience during setup/testing.
// In production, use a proper key management system.
func GenerateKey() ([]byte, error) {
	// TODO: Create 32-byte slice
	// TODO: Fill with crypto random bytes using io.ReadFull(rand.Reader, key)
	// TODO: Handle random generation error
	// TODO: Return key
	return nil, fmt.Errorf("not implemented")
}

// EncodeKeyBase64 encodes an encryption key as base64 for storage in env vars.
func EncodeKeyBase64(key []byte) string {
	// TODO: Encode key as base64 using base64.StdEncoding.EncodeToString(key)
	// TODO: Return base64 string
	return ""
}

// DecodeKeyBase64 decodes a base64-encoded encryption key from env vars.
func DecodeKeyBase64(encodedKey string) ([]byte, error) {
	// TODO: Decode base64 string using base64.StdEncoding.DecodeString(encodedKey)
	// TODO: Handle decode error
	// TODO: Validate that decoded key is exactly 32 bytes
	// TODO: If wrong length, return error: "invalid key length after base64 decode"
	// TODO: Return decoded key
	return nil, fmt.Errorf("not implemented")
}
