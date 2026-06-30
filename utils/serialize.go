package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// EncryptSource encrypts sourceBytes with AES-256-GCM.
// The encryption key is derived from secretKey via SHA-256.
func EncryptSource(sourceBytes []byte, secretKey string) (string, error) {
	keyHash := sha256.Sum256([]byte(secretKey))
	key := keyHash[:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, sourceBytes, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
