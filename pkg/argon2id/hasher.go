package argon2id

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	timeCost = 1
	memory   = 64 * 1024
	threads  = 4
	keyLen   = 32
	saltLen  = 16
)

type Hasher struct{}

func NewHasher() *Hasher {
	return &Hasher{}
}

func (h *Hasher) Hash(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	sum := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, keyLen)
	return fmt.Sprintf("argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, timeCost, threads, b64(salt), b64(sum)), nil
}

func (h *Hasher) Compare(hash, password string) error {
	parts := strings.Split(hash, "$")
	if len(parts) != 5 || parts[0] != "argon2id" {
		return fmt.Errorf("unknown hash")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return err
	}

	want, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return err
	}

	sum := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(want)))
	if subtle.ConstantTimeCompare(sum, want) != 1 {
		return fmt.Errorf("mismatch")
	}

	return nil
}

func b64(b []byte) string {
	return base64.RawStdEncoding.EncodeToString(b)
}
