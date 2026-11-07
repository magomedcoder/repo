package sshsig

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/pem"
	"errors"
	"fmt"
	"hash"
	"io"
	"strings"

	"golang.org/x/crypto/ssh"
)

const (
	Magic     = "SSHSIG"
	PEMType   = "SSH SIGNATURE"
	Namespace = "git"
)

type wrappedSig struct {
	MagicHeader   [6]byte
	Version       uint32
	PublicKey     string
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Signature     string
}

type messageWrapper struct {
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Hash          string
}

var hashAlgs = map[string]func() hash.Hash{
	"sha256": sha256.New,
	"sha512": sha512.New,
}

type Signer struct {
	Key        ssh.PublicKey
	Namespaces map[string]struct{} // empty = any
}

func ParseAllowedSigners(data []byte) ([]Signer, error) {
	var out []Signer
	for raw := range strings.SplitSeq(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		signer, err := parseAllowedSignerLine(line)
		if err != nil {
			return nil, err
		}

		out = append(out, signer)
	}

	return out, nil
}

func parseAllowedSignerLine(line string) (Signer, error) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return Signer{}, fmt.Errorf("allowed_signers: invalid line")
	}

	i := 1
	namespaces := map[string]struct{}{}
	for i < len(fields) {
		f := fields[i]
		if strings.Contains(f, "=") {
			key, val, _ := strings.Cut(f, "=")
			if key == "namespaces" {
				val = strings.Trim(val, `"`)
				for ns := range strings.SplitSeq(val, ",") {
					ns = strings.TrimSpace(ns)
					if ns != "" {
						namespaces[ns] = struct{}{}
					}
				}
			}
			i++
			continue
		}
		break
	}

	if i+1 >= len(fields) {
		return Signer{}, fmt.Errorf("allowed_signers: missing key")
	}

	keyLine := strings.Join(fields[i:], " ")
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keyLine))
	if err != nil {
		return Signer{}, fmt.Errorf("allowed_signers: %w", err)
	}

	return Signer{Key: pub, Namespaces: namespaces}, nil
}

func Verify(message io.Reader, armored string, signers []Signer) error {
	return VerifyNamespace(message, armored, signers, Namespace)
}

func VerifyNamespace(message io.Reader, armored string, signers []Signer, namespace string) error {
	if len(signers) == 0 {
		return errors.New("sshsig: no allowed signers configured")
	}

	sig, err := decode([]byte(armored))
	if err != nil {
		return err
	}

	if sig.Namespace != namespace {
		return fmt.Errorf("sshsig: namespace %q, want %q", sig.Namespace, namespace)
	}

	newHash, ok := hashAlgs[sig.HashAlgorithm]
	if !ok {
		return fmt.Errorf("sshsig: unsupported hash %q", sig.HashAlgorithm)
	}

	embedded, err := ssh.ParsePublicKey([]byte(sig.PublicKey))
	if err != nil {
		return fmt.Errorf("sshsig: public key: %w", err)
	}

	var sshSig ssh.Signature
	if err := ssh.Unmarshal([]byte(sig.Signature), &sshSig); err != nil {
		return fmt.Errorf("sshsig: signature blob: %w", err)
	}

	h := newHash()
	if _, err := io.Copy(h, message); err != nil {
		return err
	}
	hm := h.Sum(nil)

	toVerify := ssh.Marshal(messageWrapper{
		Namespace:     namespace,
		HashAlgorithm: sig.HashAlgorithm,
		Hash:          string(hm),
	})
	toVerify = append([]byte(Magic), toVerify...)

	for _, s := range signers {
		if len(s.Namespaces) > 0 {
			if _, ok := s.Namespaces[namespace]; !ok {
				continue
			}
		}

		if !bytes.Equal(s.Key.Marshal(), embedded.Marshal()) {
			continue
		}

		if err := s.Key.Verify(toVerify, &sshSig); err != nil {
			return fmt.Errorf("sshsig: invalid signature: %w", err)
		}

		return nil
	}

	return errors.New("sshsig: signing key not in allowed_signers")
}

func decode(armored []byte) (*wrappedSig, error) {
	block, _ := pem.Decode(armored)
	if block == nil {
		return nil, errors.New("sshsig: invalid pem")
	}

	if block.Type != PEMType {
		return nil, fmt.Errorf("sshsig: unexpected pem type %q", block.Type)
	}

	var sig wrappedSig
	if err := ssh.Unmarshal(block.Bytes, &sig); err != nil {
		return nil, fmt.Errorf("sshsig: unmarshal: %w", err)
	}

	if sig.Version != 1 {
		return nil, fmt.Errorf("sshsig: unsupported version %d", sig.Version)
	}

	if string(sig.MagicHeader[:]) != Magic {
		return nil, errors.New("sshsig: bad magic")
	}

	return &sig, nil
}
