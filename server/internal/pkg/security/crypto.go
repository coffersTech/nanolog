package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// MasterKey is the global 32-byte key used for data encryption.
var MasterKey []byte

// InitMasterKey initializes the master key from environment, file, or generates a new one.
// Returns (true, nil) if a new key was generated.
func InitMasterKey(keyPath string) (bool, error) {
	// 1. Check Environmental Variable
	if envKey := os.Getenv("NANOLOG_MASTER_KEY"); envKey != "" {
		key, err := hex.DecodeString(envKey)
		if err == nil && len(key) == 32 {
			MasterKey = key
			return false, nil
		}
	}

	// 2. Check Key File
	if _, err := os.Stat(keyPath); err == nil {
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return false, fmt.Errorf("failed to read key file: %w", err)
		}

		keyStr := strings.TrimSpace(string(data))
		key, err := hex.DecodeString(keyStr)
		if err == nil && len(key) == 32 {
			MasterKey = key
			return false, nil
		}
	}

	// 3. Generate New Key
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return false, fmt.Errorf("failed to generate random key: %w", err)
	}

	keyHex := hex.EncodeToString(key)
	if err := os.WriteFile(keyPath, []byte(keyHex), 0600); err != nil {
		return false, fmt.Errorf("failed to save master key to %s: %w", keyPath, err)
	}

	MasterKey = key
	return true, nil
}

// Encrypt encrypts plaintext using AES-GCM and returns Nonce + Ciphertext.
func Encrypt(plaintext []byte) ([]byte, error) {
	if len(MasterKey) != 32 {
		return nil, errors.New("master key not initialized or invalid length")
	}

	block, err := aes.NewCipher(MasterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal returns nonce + ciphertext
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext (Nonce + Ciphertext) using AES-GCM.
func Decrypt(data []byte) ([]byte, error) {
	if len(MasterKey) != 32 {
		return nil, errors.New("master key not initialized or invalid length")
	}

	block, err := aes.NewCipher(MasterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
