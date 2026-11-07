package sshsig

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/pem"
	"io"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestParseAllowedSigners(t *testing.T) {
	t.Parallel()

	raw := `# comment
* namespaces="git" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEHjnNEfE88W1pvBLdV3otv28x760gdmPao3lVD5uAt9 tip@example
alice@example.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEHjnNEfE88W1pvBLdV3otv28x760gdmPao3lVD5uAt9
`
	signers, err := ParseAllowedSigners([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}

	if len(signers) != 2 {
		t.Fatalf("got %d signers", len(signers))
	}

	if _, ok := signers[0].Namespaces["git"]; !ok {
		t.Fatal("expected git namespace on first signer")
	}

	if len(signers[1].Namespaces) != 0 {
		t.Fatal("second signer should allow any namespace")
	}
}

func TestVerifyRoundTrip(t *testing.T) {
	t.Parallel()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	rawSigner, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}

	signer, ok := rawSigner.(ssh.AlgorithmSigner)
	if !ok {
		t.Fatal("signer is not AlgorithmSigner")
	}
	authorized := string(ssh.MarshalAuthorizedKey(signer.PublicKey()))

	signers, err := ParseAllowedSigners([]byte("* namespaces=\"git\" " + strings.TrimSpace(authorized) + "\n"))
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("payload\n")
	armored, err := signGit(signer, bytes.NewReader(msg))
	if err != nil {
		t.Fatal(err)
	}

	if err := Verify(bytes.NewReader(msg), armored, signers); err != nil {
		t.Fatal(err)
	}

	if err := Verify(bytes.NewReader([]byte("tampered")), armored, signers); err == nil {
		t.Fatal("expected tamper failure")
	}
}

func signGit(s ssh.AlgorithmSigner, m io.Reader) (string, error) {
	const hashAlg = "sha512"
	h := sha512.New()
	if _, err := io.Copy(h, m); err != nil {
		return "", err
	}
	hm := h.Sum(nil)

	toSign := ssh.Marshal(messageWrapper{
		Namespace:     Namespace,
		HashAlgorithm: hashAlg,
		Hash:          string(hm),
	})
	toSign = append([]byte(Magic), toSign...)

	sig, err := s.SignWithAlgorithm(rand.Reader, toSign, "")
	if err != nil {
		return "", err
	}

	wrapped := wrappedSig{
		Version:       1,
		PublicKey:     string(s.PublicKey().Marshal()),
		Namespace:     Namespace,
		HashAlgorithm: hashAlg,
		Signature:     string(ssh.Marshal(sig)),
	}
	copy(wrapped.MagicHeader[:], Magic)

	block := &pem.Block{
		Type:  PEMType,
		Bytes: ssh.Marshal(&wrapped),
	}

	return string(pem.EncodeToMemory(block)), nil
}
