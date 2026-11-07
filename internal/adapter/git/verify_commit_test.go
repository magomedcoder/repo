package git

import (
	"bytes"
	"crypto"
	"strings"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

func TestVerifyCommitObjectUnsigned(t *testing.T) {
	t.Parallel()

	r := &Repository{}
	err := r.verifyCommitObject(&object.Commit{
		Hash:    plumbing.NewHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		Message: "x\n",
		Author: object.Signature{
			Name:  "a",
			Email: "a@b.c",
			When:  time.Unix(1, 0),
		},
		Committer: object.Signature{
			Name:  "a",
			Email: "a@b.c",
			When:  time.Unix(1, 0),
		},
	})
	if err == nil || !strings.Contains(err.Error(), "not signed") {
		t.Fatalf("got %v", err)
	}
}

func TestVerifyCommitObjectOpenPGP(t *testing.T) {
	t.Parallel()

	entity, err := openpgp.NewEntity("Tip", "", "tip@example.com", &packet.Config{
		DefaultHash: crypto.SHA256,
		RSABits:     2048,
	})
	if err != nil {
		t.Fatal(err)
	}

	var pubBuf bytes.Buffer
	w, err := armor.Encode(&pubBuf, openpgp.PublicKeyType, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := entity.Serialize(w); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	commit := &object.Commit{
		Message: "signed tip\n",
		Author: object.Signature{
			Name:  "Tip",
			Email: "tip@example.com",
			When:  time.Unix(1_700_000_000, 0),
		},
		Committer: object.Signature{
			Name:  "Tip",
			Email: "tip@example.com",
			When:  time.Unix(1_700_000_000, 0),
		},
		TreeHash: plumbing.NewHash("4b825dc642cb6eb9a060e54bf8d69288fbee4904"),
	}

	payload := &plumbing.MemoryObject{}
	if err := commit.EncodeWithoutSignature(payload); err != nil {
		t.Fatal(err)
	}

	pr, err := payload.Reader()
	if err != nil {
		t.Fatal(err)
	}

	var sigBuf bytes.Buffer
	if err := openpgp.ArmoredDetachSign(&sigBuf, entity, pr, nil); err != nil {
		t.Fatal(err)
	}
	commit.Signature = sigBuf.String()

	r := &Repository{trust: TrustMaterial{
		OpenPGPKeyring: pubBuf.String(),
	}}
	if err := r.verifyCommitObject(commit); err != nil {
		t.Fatal(err)
	}

	r.trust.OpenPGPKeyring = ""
	if err := r.verifyCommitObject(commit); err == nil {
		t.Fatal("expected missing keyring error")
	}
}
