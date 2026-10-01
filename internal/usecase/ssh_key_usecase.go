package usecase

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"golang.org/x/crypto/ssh"
)

var (
	ErrSSHKeyTitleRequired = errors.New("ssh_key_title_required")
	ErrSSHKeyRequired      = errors.New("ssh_key_required")
	ErrInvalidSSHKey       = errors.New("invalid_ssh_key")
	ErrSSHKeyExists        = errors.New("ssh_key_already_exists")
	ErrSSHKeyNotFound      = errors.New("ssh_key_not_found")
	ErrInvalidSSHKeyID     = errors.New("invalid_ssh_key_id")
)

type SSHKeyItem struct {
	ID          uint       `json:"id"`
	Title       string     `json:"title"`
	Fingerprint string     `json:"fingerprint"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

type SSHKeyUseCase struct {
	keys  domain.SSHKeyStore
	users domain.UserStore
}

func NewSSHKeyUseCase(keys domain.SSHKeyStore, users domain.UserStore) *SSHKeyUseCase {
	return &SSHKeyUseCase{
		keys:  keys,
		users: users,
	}
}

func (uc *SSHKeyUseCase) Create(userID uint, title, publicKey string) (*SSHKeyItem, error) {
	if userID == 0 {
		return nil, ErrUnauthorized
	}

	normalized, fingerprint, comment, err := normalizePublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	title = strings.TrimSpace(title)
	if title == "" {
		title = strings.TrimSpace(comment)
	}

	if title == "" {
		title = "SSH key"
	}

	if len(title) > 100 {
		title = title[:100]
	}

	if existing, err := uc.keys.FindByFingerprint(fingerprint); err == nil && existing != nil {
		return nil, ErrSSHKeyExists
	} else if err != nil && !isNotFound(err) {
		return nil, fmt.Errorf("check ssh key: %w", err)
	}

	key := &domain.SSHKey{
		UserID:      userID,
		Title:       title,
		PublicKey:   normalized,
		Fingerprint: fingerprint,
	}

	if err := uc.keys.Create(key); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, ErrSSHKeyExists
		}

		return nil, fmt.Errorf("save ssh key: %w", err)
	}
	item := toSSHKeyItem(key)

	return &item, nil
}

func (uc *SSHKeyUseCase) List(userID uint) ([]SSHKeyItem, error) {
	if userID == 0 {
		return nil, ErrUnauthorized
	}

	keys, err := uc.keys.ListByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("list ssh keys: %w", err)
	}

	items := make([]SSHKeyItem, 0, len(keys))
	for i := range keys {
		items = append(items, toSSHKeyItem(&keys[i]))
	}

	return items, nil
}

func (uc *SSHKeyUseCase) Delete(userID, keyID uint) error {
	if userID == 0 {
		return ErrUnauthorized
	}

	if keyID == 0 {
		return ErrInvalidSSHKeyID
	}

	if err := uc.keys.DeleteByUserAndID(userID, keyID); err != nil {
		if isNotFound(err) {
			return ErrSSHKeyNotFound
		}
		return fmt.Errorf("delete ssh key: %w", err)
	}

	return nil
}

func (uc *SSHKeyUseCase) AuthenticateFingerprint(fingerprint string) (*domain.User, error) {
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return nil, ErrUnauthorized
	}

	key, err := uc.keys.FindByFingerprint(fingerprint)
	if err != nil {
		return nil, ErrUnauthorized
	}

	user, err := uc.users.FindByID(key.UserID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	_ = uc.keys.TouchLastUsed(key.ID, time.Now())

	return sanitizeUser(user), nil
}

func normalizePublicKey(raw string) (normalized, fingerprint, comment string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", ErrSSHKeyRequired
	}

	pub, comment, _, rest, parseErr := ssh.ParseAuthorizedKey([]byte(raw))
	if parseErr != nil || pub == nil {
		return "", "", "", ErrInvalidSSHKey
	}

	if len(strings.TrimSpace(string(rest))) > 0 {
		return "", "", "", ErrInvalidSSHKey
	}

	normalized = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
	fingerprint = ssh.FingerprintSHA256(pub)
	return normalized, fingerprint, comment, nil
}

func toSSHKeyItem(key *domain.SSHKey) SSHKeyItem {
	return SSHKeyItem{
		ID:          key.ID,
		Title:       key.Title,
		Fingerprint: key.Fingerprint,
		CreatedAt:   key.CreatedAt,
		LastUsedAt:  key.LastUsedAt,
	}
}
