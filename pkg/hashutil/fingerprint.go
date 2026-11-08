package hashutil

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func ShortSHA256(input string, nBytes int) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	if nBytes <= 0 {
		nBytes = 8
	}

	if nBytes > sha256.Size {
		nBytes = sha256.Size
	}

	sum := sha256.Sum256([]byte(input))

	return hex.EncodeToString(sum[:nBytes])
}

func LoginFingerprint(login string) string {
	return ShortSHA256(strings.ToLower(strings.TrimSpace(login)), 8)
}
