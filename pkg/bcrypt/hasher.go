package bcrypt

import (
	"golang.org/x/crypto/bcrypt"
)

type Hasher struct {
	cost int
}

func NewHasher() *Hasher {
	return &Hasher{cost: bcrypt.DefaultCost}
}

func (h *Hasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (h *Hasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
