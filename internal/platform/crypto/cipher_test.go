package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	cipher, err := NewCipher(key)
	require.NoError(t, err)

	plaintext := "xoxb-111-222-supersecret"
	ciphertext, err := cipher.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := cipher.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_NonDeterministic(t *testing.T) {
	key := make([]byte, 32)
	cipher, _ := NewCipher(key)

	c1, _ := cipher.Encrypt("same-value")
	c2, _ := cipher.Encrypt("same-value")
	assert.NotEqual(t, c1, c2, "each encryption must use a fresh random nonce")
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 0xFF

	c1, _ := NewCipher(key1)
	c2, _ := NewCipher(key2)

	ciphertext, _ := c1.Encrypt("secret")
	_, err := c2.Decrypt(ciphertext)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	c, _ := NewCipher(key)

	ciphertext, _ := c.Encrypt("secret")
	tampered := ciphertext[:len(ciphertext)-4] + "XXXX"

	_, err := c.Decrypt(tampered)
	assert.Error(t, err)
}

func TestEncryptDecrypt_EmptyString(t *testing.T) {
	key := make([]byte, 32)
	c, _ := NewCipher(key)

	enc, _ := c.Encrypt("")
	assert.Equal(t, "", enc, "empty plaintext should produce empty ciphertext")

	dec, _ := c.Decrypt("")
	assert.Equal(t, "", dec, "empty ciphertext should produce empty plaintext")
}
