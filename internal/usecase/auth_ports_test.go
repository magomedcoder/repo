package usecase

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/attempts"
)

type memUsers struct {
	mu    sync.Mutex
	items []*domain.User
	next  uint
}

func (s *memUsers) Create(user *domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	user.ID = s.next
	cp := *user
	s.items = append(s.items, &cp)
	return nil
}

func (s *memUsers) FindByID(id uint) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.items {
		if u.ID == id {
			cp := *u
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (s *memUsers) FindByIDs(ids []uint) (map[uint]*domain.User, error) {
	out := make(map[uint]*domain.User)
	if len(ids) == 0 {
		return out, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		for _, u := range s.items {
			if u.ID == id {
				cp := *u
				out[id] = &cp
				break
			}
		}
	}
	return out, nil
}

func (s *memUsers) FindByUsername(username string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.items {
		if u.Username == username {
			cp := *u
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (s *memUsers) FindByEmail(email string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.items {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (s *memUsers) ExistsByUsername(username string) (bool, error) {
	_, err := s.FindByUsername(username)
	return err == nil, nil
}

func (s *memUsers) ExistsByEmail(email string) (bool, error) {
	_, err := s.FindByEmail(email)
	return err == nil, nil
}

func (s *memUsers) Update(user *domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, u := range s.items {
		if u.ID == user.ID {
			cp := *user
			s.items[i] = &cp
			return nil
		}
	}
	return errors.New("not found")
}

func (s *memUsers) Delete(id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, u := range s.items {
		if u.ID == id {
			s.items = append(s.items[:i], s.items[i+1:]...)
			return nil
		}
	}

	return errors.New("not found")
}

func (s *memUsers) List(filter domain.UserListFilter) ([]domain.User, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]domain.User, 0, len(s.items))
	for _, u := range s.items {
		out = append(out, *u)
	}

	total := len(out)
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	offset := min(filter.Offset, total)

	end := min(offset+limit, total)

	return out[offset:end], total, nil
}

func (s *memUsers) CountByRole(role domain.Role) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !role.Valid() {
		role = domain.RoleUser
	}

	var n int64
	for _, u := range s.items {
		r := u.Role
		if !r.Valid() {
			r = domain.RoleUser
		}

		if r == role {
			n++
		}
	}

	return n, nil
}

type memSessions struct {
	mu    sync.Mutex
	items []*domain.Session
}

func (s *memSessions) Create(session *domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *session
	if cp.ID == 0 {
		cp.ID = uint(len(s.items) + 1)
		session.ID = cp.ID
	}
	s.items = append(s.items, &cp)
	return nil
}

func (s *memSessions) FindByToken(token string) (*domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.items {
		if item.Token == token {
			cp := *item
			return &cp, nil
		}
	}

	return nil, errors.New("not found")
}

func (s *memSessions) ListByUserID(userID uint) ([]domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.Session
	for _, item := range s.items {
		if item.UserID == userID {
			out = append(out, *item)
		}
	}

	return out, nil
}

func (s *memSessions) DeleteByUserAndID(userID, sessionID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := s.items[:0]
	found := false
	for _, item := range s.items {
		if item.UserID == userID && item.ID == sessionID {
			found = true
			continue
		}
		out = append(out, item)
	}

	s.items = out
	if !found {
		return domain.ErrNotFound
	}

	return nil
}

func (s *memSessions) Touch(session *domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.items {
		if item.ID == session.ID || item.Token == session.Token {
			item.LastSeenAt = session.LastSeenAt
			item.ExpiresAt = session.ExpiresAt
			return nil
		}
	}
	return nil
}

func (s *memSessions) DeleteByToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.items[:0]
	for _, item := range s.items {
		if item.Token != token {
			out = append(out, item)
		}
	}
	s.items = out
	return nil
}

func (s *memSessions) DeleteByUserIDExcept(userID uint, keepToken string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.items[:0]
	for _, item := range s.items {
		if item.UserID == userID && item.Token != keepToken {
			continue
		}
		out = append(out, item)
	}
	s.items = out
	return nil
}

func (s *memSessions) DeleteByUserID(userID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := s.items[:0]
	for _, item := range s.items {
		if item.UserID == userID {
			continue
		}

		out = append(out, item)
	}
	s.items = out

	return nil
}

type memTokens struct {
	n int
}

func (t *memTokens) Generate() (string, error) {
	t.n++
	return fmt.Sprintf("token-%d", t.n), nil
}

type memAccessTokens struct {
	deletedFor []uint
}

func (m *memAccessTokens) Create(*domain.AccessToken) error {
	return nil
}

func (m *memAccessTokens) ListByUserID(uint) ([]domain.AccessToken, error) {
	return nil, nil
}

func (m *memAccessTokens) FindByHash(string) (*domain.AccessToken, error) {
	return nil, errors.New("not found")
}

func (m *memAccessTokens) DeleteByUserAndID(uint, uint) error {
	return nil
}

func (m *memAccessTokens) DeleteByUserID(id uint) error {
	m.deletedFor = append(m.deletedFor, id)
	return nil
}

func (m *memAccessTokens) TouchLastUsed(uint, time.Time) error {
	return nil
}

type memHasher struct{}

func (memHasher) Hash(password string) (string, error) {
	return "h:" + password, nil
}

func (memHasher) Compare(hash, password string) error {
	if hash == "h:"+password {
		return nil
	}

	return errors.New("mismatch")
}

func testAuth(users *memUsers, sessions domain.SessionStore) *AuthUseCase {
	uc := NewAuthUseCase(users, sessions, &memAccessTokens{}, memHasher{}, &memTokens{})
	uc.SetSessionSecret("0123456789abcdef0123456789abcdef")
	return uc
}

func TestRegisterClosed(t *testing.T) {
	t.Parallel()
	uc := testAuth(&memUsers{}, &memSessions{})
	_, err := uc.Register(t.Context(), RegisterInput{Username: "ada", Email: "ada@example.com", Password: "password1"})
	if !errors.Is(err, ErrRegistrationClosed) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateUser(t *testing.T) {
	t.Parallel()
	users := &memUsers{}
	uc := testAuth(users, &memSessions{})
	user, err := uc.CreateUser(t.Context(), RegisterInput{Username: "ada", Email: "ada@example.com", Password: "password1"})
	if err != nil {
		t.Fatal(err)
	}

	if user.Username != "ada" || user.ID == 0 {
		t.Fatalf("%+v", user)
	}

	_, err = uc.CreateUser(t.Context(), RegisterInput{Username: "ada", Email: "other@example.com", Password: "password1"})
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("got %v", err)
	}
}

func TestChangePasswordRevokesOtherSessions(t *testing.T) {
	t.Parallel()
	users := &memUsers{}
	sessions := &memSessions{}
	uc := testAuth(users, sessions)
	at := &memAccessTokens{}
	uc.accessTokens = at
	user, err := uc.CreateUser(t.Context(), RegisterInput{Username: "ada", Email: "ada@example.com", Password: "password1"})
	if err != nil {
		t.Fatal(err)
	}

	keep, err := uc.createSession(user.ID, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}

	other, err := uc.createSession(user.ID, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}

	if err := uc.ChangePassword(t.Context(), ChangePasswordInput{
		UserID:    user.ID,
		Current:   "password1",
		Next:      "password2",
		KeepToken: keep,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := sessions.FindByToken(keep); err != nil {
		t.Fatal("kept session missing")
	}

	if _, err := sessions.FindByToken(other); err == nil {
		t.Fatal("other session still present")
	}

	if len(at.deletedFor) != 1 || at.deletedFor[0] != user.ID {
		t.Fatalf("access tokens not revoked: %v", at.deletedFor)
	}
}

func TestAttemptLockExpires(t *testing.T) {
	t.Parallel()
	now := time.Now()
	m := attempts.New(attempts.Config{
		MaxFails: 5,
		BaseLock: time.Minute,
		Now:      func() time.Time { return now },
	})

	for range 5 {
		m.Fail("k")
	}

	if !m.Locked("k") {
		t.Fatal("expected lock")
	}

	now = now.Add(2 * time.Minute)
	if m.Locked("k") {
		t.Fatal("expected unlock after ttl")
	}
}
