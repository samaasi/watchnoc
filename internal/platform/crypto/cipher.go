package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Cipher performs AES-256-GCM authenticated encryption.
// The nonce (12 bytes) is prepended to the ciphertext before base64 encoding.
// Stored format: ENC[v1]base64(nonce || ciphertext || tag)
type Cipher struct {
	gcm cipher.AEAD
}

const cipherPrefix = "ENC[v1]"

// NewCipher initialises a Cipher from a 32-byte AES-256 key.
// key must be exactly 32 bytes. Load from environment / AWS Secrets Manager —
// never hardcode.
func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: create GCM: %w", err)
	}
	return &Cipher{gcm: gcm}, nil
}

// Encrypt encrypts plaintext and returns a prefixed base64-encoded string suitable for
// storage in a VARCHAR/TEXT database column.
//
// Each call produces a unique ciphertext because a fresh random nonce is used —
// encrypting the same plaintext twice will yield different results.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil // Empty strings are stored as empty — not as ciphertext
	}

	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: generate nonce: %w", err)
	}

	ciphertext := c.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return cipherPrefix + encoded, nil
}

// Decrypt decrypts a prefixed base64-encoded ciphertext produced by Encrypt.
// Returns ErrDecryptionFailed if the ciphertext is invalid or tampered with.
func (c *Cipher) Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	if !strings.HasPrefix(encoded, cipherPrefix) {
		return "", ErrDecryptionFailed
	}

	b64Data := strings.TrimPrefix(encoded, cipherPrefix)
	data, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return "", fmt.Errorf("crypto: base64 decode: %w", err)
	}

	nonceSize := c.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrDecryptionFailed
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// GCM authentication failure — ciphertext tampered or wrong key
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// ErrDecryptionFailed is returned when GCM authentication fails.
// This means either the ciphertext was tampered with, or the wrong key is in use.
// NEVER log the ciphertext value alongside this error.
var ErrDecryptionFailed = errors.New("crypto: decryption failed — invalid ciphertext or wrong key")
