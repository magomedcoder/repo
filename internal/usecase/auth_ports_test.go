package usecase

import (
	"errors"
	"testing"

	"git.magomedcoder.ru/repo/internal/domain"
)

type memUsers struct {
	items []*domain.User
	next  uint
}

func (s *memUsers) Create(user *domain.User) error {
	s.next++
	user.ID = s.next
	cp := *user
	s.items = append(s.items, &cp)
	return nil
}

func (s *memUsers) FindByID(id uint) (*domain.User, error) {
	for _, u := range s.items {
		if u.ID == id {
			cp := *u
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (s *memUsers) FindByUsername(username string) (*domain.User, error) {
	for _, u := range s.items {
		if u.Username == username {
			cp := *u
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}

func (s *memUsers) FindByEmail(email string) (*domain.User, error) {
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

func (s *memUsers) Count() (int64, error) {
	return int64(len(s.items)), nil
}

func (s *memUsers) Update(user *domain.User) error {
	for i, u := range s.items {
		if u.ID == user.ID {
			cp := *user
			s.items[i] = &cp
			return nil
		}
	}
	return errors.New("not found")
}

type memSessions struct{}

func (memSessions) Create(*domain.Session) error {
	return nil
}

func (memSessions) FindByToken(string) (*domain.Session, error) {
	return nil, errors.New("not found")
}

func (memSessions) DeleteByToken(string) error {
	return nil
}

type memTokens struct{}

func (memTokens) Generate() (string, error) {
	return "token", nil
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

func TestEnsureOwnerClosesRegistration(t *testing.T) {
	users := &memUsers{}
	uc := NewAuthUseCase(users, memSessions{}, memHasher{}, memTokens{})
	if err := uc.EnsureOwner("ada", "ada@example.com", "password1"); err != nil {
		t.Fatal(err)
	}

	_, err := uc.Register(RegisterInput{Username: "grace", Email: "grace@example.com", Password: "password1"})
	if !errors.Is(err, ErrRegistrationClosed) {
		t.Fatalf("second register: %v", err)
	}
}
