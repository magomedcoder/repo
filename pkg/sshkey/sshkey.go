package sshkey

import (
	"errors"
	"strings"

	gossh "golang.org/x/crypto/ssh"
)

var ErrInvalid = errors.New("invalid ssh private key")

func ValidatePrivateKey(key, passphrase string) error {
	if strings.TrimSpace(key) == "" {
		return nil
	}

	_, err := gossh.ParsePrivateKey([]byte(key))
	if _, missing := err.(*gossh.PassphraseMissingError); missing {
		_, err = gossh.ParsePrivateKeyWithPassphrase([]byte(key), []byte(passphrase))
	}

	if err != nil {
		return ErrInvalid
	}

	return nil
}
