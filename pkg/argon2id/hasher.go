package argon2id

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	timeCost = 2
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

	params, err := parseParams(parts[2])
	if err != nil {
		return err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return err
	}

	want, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return err
	}

	sum := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(want)))
	if subtle.ConstantTimeCompare(sum, want) != 1 {
		return fmt.Errorf("mismatch")
	}

	return nil
}

func (h *Hasher) NeedsRehash(hash string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 5 || parts[0] != "argon2id" {
		return true
	}

	params, err := parseParams(parts[2])
	if err != nil {
		return true
	}

	return params.time < timeCost || params.memory < memory || params.threads != threads
}

type argonParams struct {
	memory  uint32
	time    uint32
	threads uint8
}

func parseParams(raw string) (argonParams, error) {
	var out argonParams
	for part := range strings.SplitSeq(raw, ",") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			return out, fmt.Errorf("bad params")
		}

		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return out, err
		}

		switch k {
		case "m":
			out.memory = uint32(n)
		case "t":
			out.time = uint32(n)
		case "p":
			out.threads = uint8(n)
		}
	}

	if out.memory == 0 || out.time == 0 || out.threads == 0 {
		return out, fmt.Errorf("incomplete params")
	}

	return out, nil
}

func b64(b []byte) string {
	return base64.RawStdEncoding.EncodeToString(b)
}
