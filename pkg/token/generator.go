package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

type Generator struct {
	bytes int
}

func NewGenerator() *Generator {
	return &Generator{bytes: 32}
}

func (g *Generator) Generate() (string, error) {
	buf := make([]byte, g.bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}
