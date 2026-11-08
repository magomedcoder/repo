package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"magomedcoder.ru/repo/pkg/fsutil"
	"magomedcoder.ru/repo/pkg/sshsig"
)

type TrustMaterial struct {
	OpenPGPKeyring string
	AllowedSigners []sshsig.Signer
}

func (t TrustMaterial) Empty() bool {
	return strings.TrimSpace(t.OpenPGPKeyring) == "" && len(t.AllowedSigners) == 0
}

func (r *Repository) SetTrust(t TrustMaterial) {
	r.trust = t
}

func (r *Repository) VerifyTrustConfigured() bool {
	return !r.trust.Empty()
}

func LoadTrustFiles(openpgpPath, allowedSignersPath string) (TrustMaterial, error) {
	var t TrustMaterial

	if p := strings.TrimSpace(openpgpPath); p != "" {
		raw, err := os.ReadFile(p)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return TrustMaterial{}, fmt.Errorf("mirror verify openpgp keyring: %w", err)
			}
		} else {
			t.OpenPGPKeyring = string(raw)
		}
	}

	if p := strings.TrimSpace(allowedSignersPath); p != "" {
		raw, err := os.ReadFile(p)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return TrustMaterial{}, fmt.Errorf("mirror verify allowed_signers: %w", err)
			}
		} else {
			signers, err := sshsig.ParseAllowedSigners(raw)
			if err != nil {
				return TrustMaterial{}, err
			}

			t.AllowedSigners = signers
		}
	}

	return t, nil
}

func ResolveTrustPath(dataDir, p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", nil
	}

	if filepath.IsAbs(p) {
		return "", fmt.Errorf("trust path %q must be relative to data_dir", p)
	}

	joined := filepath.Join(dataDir, filepath.Clean(p))
	safe, err := fsutil.SafePath(dataDir, joined)
	if err != nil {
		return "", fmt.Errorf("trust path: %w", err)
	}

	return safe, nil
}

func (r *Repository) VerifyCommitSignature(ctx context.Context, repoPath, sha string) error {
	if err := checkCtx(ctx); err != nil {
		return err
	}

	sha = strings.TrimSpace(sha)
	if sha == "" {
		return fmt.Errorf("verify-commit: empty sha")
	}

	repo, err := r.open(repoPath)
	if err != nil {
		return err
	}

	commit, err := repo.CommitObject(plumbing.NewHash(sha))
	if err != nil {
		return fmt.Errorf("verify-commit: %w", err)
	}

	return r.verifyCommitObject(commit)
}

func (r *Repository) verifyCommitObject(commit *object.Commit) error {
	if commit == nil {
		return fmt.Errorf("verify-commit: nil commit")
	}

	sig := strings.TrimSpace(commit.Signature)
	if sig == "" {
		sig = strings.TrimSpace(commit.SignatureSHA256)
	}

	if sig == "" {
		return errors.New("verify-commit: commit is not signed")
	}

	switch {
	case strings.Contains(sig, "BEGIN SSH SIGNATURE"):
		encoded := &plumbing.MemoryObject{}
		if err := commit.EncodeWithoutSignature(encoded); err != nil {
			return fmt.Errorf("verify-commit: payload: %w", err)
		}

		reader, err := encoded.Reader()
		if err != nil {
			return err
		}
		defer reader.Close()
		if err := sshsig.Verify(reader, sig, r.trust.AllowedSigners); err != nil {
			return fmt.Errorf("verify-commit: %w", err)
		}

		return nil

	case strings.Contains(sig, "BEGIN PGP SIGNATURE"), strings.Contains(sig, "BEGIN PGP MESSAGE"):
		if strings.TrimSpace(r.trust.OpenPGPKeyring) == "" {
			return errors.New("verify-commit: no OpenPGP keyring configured")
		}

		if _, err := commit.Verify(r.trust.OpenPGPKeyring); err != nil {
			return fmt.Errorf("verify-commit: openpgp: %w", err)
		}

		return nil

	default:
		return errors.New("verify-commit: unsupported signature type")
	}
}
