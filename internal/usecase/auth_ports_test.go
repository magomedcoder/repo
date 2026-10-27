package usecase

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"git.magomedcoder.ru/repo/internal/domain"
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

type memSessions struct {
	mu    sync.Mutex
	items []*domain.Session
}

func (s *memSessions) Create(session *domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *session
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

type memTokens struct {
	n int
}

func (t *memTokens) Generate() (string, error) {
	t.n++
	return fmt.Sprintf("token-%d", t.n), nil
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
	uc := NewAuthUseCase(users, sessions, memHasher{}, &memTokens{})
	uc.SetSessionSecret("0123456789abcdef0123456789abcdef")
	return uc
}

func TestRegisterClosed(t *testing.T) {
	uc := testAuth(&memUsers{}, &memSessions{})
	_, err := uc.Register(RegisterInput{Username: "ada", Email: "ada@example.com", Password: "password1"})
	if !errors.Is(err, ErrRegistrationClosed) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateUser(t *testing.T) {
	users := &memUsers{}
	uc := testAuth(users, &memSessions{})
	user, err := uc.CreateUser(RegisterInput{Username: "ada", Email: "ada@example.com", Password: "password1"})
	if err != nil {
		t.Fatal(err)
	}

	if user.Username != "ada" || user.ID == 0 {
		t.Fatalf("%+v", user)
	}

	_, err = uc.CreateUser(RegisterInput{Username: "ada", Email: "other@example.com", Password: "password1"})
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("got %v", err)
	}
}

func TestChangePasswordRevokesOtherSessions(t *testing.T) {
	users := &memUsers{}
	sessions := &memSessions{}
	uc := testAuth(users, sessions)
	user, err := uc.CreateUser(RegisterInput{Username: "ada", Email: "ada@example.com", Password: "password1"})
	if err != nil {
		t.Fatal(err)
	}

	keep, err := uc.createSession(user.ID)
	if err != nil {
		t.Fatal(err)
	}

	other, err := uc.createSession(user.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := uc.ChangePassword(ChangePasswordInput{
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
}

func TestAttemptLockExpires(t *testing.T) {
	now := time.Now()
	m := &memoryAttempts{
		fails:   map[string]*attemptState{},
		now:     func() time.Time { return now },
		maxFail: 5,
		base:    time.Minute,
	}

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
