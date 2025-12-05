package crypto

import (
	"bytes"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() failed: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("GenerateKey() returned %d bytes, want 32", len(key))
	}

	// Test that consecutive keys are different
	key2, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() second call failed: %v", err)
	}
	if bytes.Equal(key, key2) {
		t.Error("GenerateKey() returned identical keys, should be random")
	}
}

func TestEncodeDecodeKeyBase64(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() failed: %v", err)
	}

	encoded := EncodeKeyBase64(key)
	if encoded == "" {
		t.Error("EncodeKeyBase64() returned empty string")
	}

	decoded, err := DecodeKeyBase64(encoded)
	if err != nil {
		t.Fatalf("DecodeKeyBase64() failed: %v", err)
	}

	if !bytes.Equal(key, decoded) {
		t.Error("DecodeKeyBase64() returned different key than original")
	}
}

func TestDecodeKeyBase64_InvalidLength(t *testing.T) {
	// Create a 16-byte key (wrong size)
	shortKey := make([]byte, 16)
	encoded := EncodeKeyBase64(shortKey)

	_, err := DecodeKeyBase64(encoded)
	if err == nil {
		t.Error("DecodeKeyBase64() should fail for non-32-byte key")
	}
}

func TestDecodeKeyBase64_InvalidBase64(t *testing.T) {
	_, err := DecodeKeyBase64("not-valid-base64!!!")
	if err == nil {
		t.Error("DecodeKeyBase64() should fail for invalid base64")
	}
}

func TestNewAESEncryptor(t *testing.T) {
	key, _ := GenerateKey()

	enc, err := NewAESEncryptor(key)
	if err != nil {
		t.Fatalf("NewAESEncryptor() failed: %v", err)
	}
	if enc == nil {
		t.Error("NewAESEncryptor() returned nil encryptor")
	}
}

func TestNewAESEncryptor_NilKey(t *testing.T) {
	_, err := NewAESEncryptor(nil)
	if err == nil {
		t.Error("NewAESEncryptor() should fail with nil key")
	}
}

func TestNewAESEncryptor_WrongKeySize(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
	}{
		{"16 bytes", 16},
		{"24 bytes", 24},
		{"31 bytes", 31},
		{"33 bytes", 33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			_, err := NewAESEncryptor(key)
			if err == nil {
				t.Errorf("NewAESEncryptor() should fail with %d byte key", tt.keySize)
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key, _ := GenerateKey()
	enc, _ := NewAESEncryptor(key)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"API key", "stripe_sk_test_12345_secret_api_key"},
		{"Empty string", ""},
		{"Unicode", "Hello 世界 🔒"},
		{"Long text", "This is a longer piece of text that should still encrypt and decrypt correctly without any issues."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plaintext := []byte(tt.plaintext)

			ciphertext, err := enc.Encrypt(plaintext)
			if err != nil {
				t.Fatalf("Encrypt() failed: %v", err)
			}

			if len(ciphertext) == 0 {
				t.Error("Encrypt() returned empty ciphertext")
			}

			decrypted, err := enc.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() failed: %v", err)
			}

			if !bytes.Equal(plaintext, decrypted) {
				t.Errorf("Decrypt() = %q, want %q", decrypted, plaintext)
			}
		})
	}
}

func TestEncrypt_NonDeterministic(t *testing.T) {
	key, _ := GenerateKey()
	enc, _ := NewAESEncryptor(key)
	plaintext := []byte("test")

	ciphertext1, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	ciphertext2, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() second call failed: %v", err)
	}

	// Ciphertexts should be different due to random nonce
	if bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("Encrypt() returned identical ciphertexts, should use random nonce")
	}

	// But both should decrypt to same plaintext
	decrypted1, _ := enc.Decrypt(ciphertext1)
	decrypted2, _ := enc.Decrypt(ciphertext2)
	if !bytes.Equal(decrypted1, decrypted2) {
		t.Error("Different ciphertexts decrypted to different plaintexts")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1, _ := GenerateKey()
	key2, _ := GenerateKey()

	enc1, _ := NewAESEncryptor(key1)
	enc2, _ := NewAESEncryptor(key2)

	plaintext := []byte("secret data")
	ciphertext, _ := enc1.Encrypt(plaintext)

	_, err := enc2.Decrypt(ciphertext)
	if err == nil {
		t.Error("Decrypt() should fail with wrong key")
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	key, _ := GenerateKey()
	enc, _ := NewAESEncryptor(key)

	_, err := enc.Decrypt([]byte("not-valid-base64!!!"))
	if err == nil {
		t.Error("Decrypt() should fail with invalid base64")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	key, _ := GenerateKey()
	enc, _ := NewAESEncryptor(key)

	// Create a valid base64 string that's too short
	shortData := EncodeKeyBase64([]byte("short"))

	_, err := enc.Decrypt([]byte(shortData))
	if err == nil {
		t.Error("Decrypt() should fail with ciphertext shorter than nonce size")
	}
}

func TestDecrypt_Tampered(t *testing.T) {
	key, _ := GenerateKey()
	enc, _ := NewAESEncryptor(key)

	plaintext := []byte("secret data")
	ciphertext, _ := enc.Encrypt(plaintext)

	// Tamper with the ciphertext by changing one character
	tampered := []byte(string(ciphertext))
	if tampered[10] == 'A' {
		tampered[10] = 'B'
	} else {
		tampered[10] = 'A'
	}

	_, err := enc.Decrypt(tampered)
	if err == nil {
		t.Error("Decrypt() should fail with tampered ciphertext (GCM authentication)")
	}
}
