package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/hkdf"
)

const (
	prefix   = "v1:"
	keyLen   = 32
	hkdfInfo = "repo-mirror-secrets-v1"
)

func DeriveKey(secret string) ([]byte, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("secret too short")
	}

	r := hkdf.New(sha256.New, []byte(secret), nil, []byte(hkdfInfo))
	key := make([]byte, keyLen)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}

	return key, nil
}

func Seal(key []byte, plain string) (string, error) {
	if plain == "" {
		return "", nil
	}

	if len(key) != keyLen {
		return "", fmt.Errorf("invalid key length")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	out := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return prefix + base64.RawStdEncoding.EncodeToString(out), nil
}

func Open(key []byte, stored string) (string, error) {
	if stored == "" {
		return "", nil
	}

	if !strings.HasPrefix(stored, prefix) {
		return stored, nil
	}

	if len(key) != keyLen {
		return "", fmt.Errorf("invalid key length")
	}

	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(stored, prefix))
	if err != nil {
		return "", fmt.Errorf("decode secret: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	plain, err := gcm.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}

	return string(plain), nil
}

func IsSealed(stored string) bool {
	return strings.HasPrefix(stored, prefix)
}
