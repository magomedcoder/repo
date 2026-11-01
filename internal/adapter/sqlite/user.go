package sqlite

import (
	"strings"
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type userModel struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt
	Username     string      `gorm:"size:39;not null"`
	Email        string      `gorm:"not null"`
	PasswordHash string      `gorm:"not null"`
	AvatarPath   string      `gorm:"size:255"`
	Role         domain.Role `gorm:"not null;default:0"`
}

func (userModel) TableName() string {
	return "users"
}

func (m userModel) toDomain() *domain.User {
	role := m.Role
	if !role.Valid() {
		role = domain.RoleUser
	}
	return &domain.User{
		ID:           m.ID,
		Username:     m.Username,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		AvatarPath:   m.AvatarPath,
		Role:         role,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

type sessionModel struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	Token      string    `gorm:"uniqueIndex;size:128;not null"`
	UserID     uint      `gorm:"index;not null;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User       userModel `gorm:"foreignKey:UserID;references:ID"`
	ExpiresAt  time.Time
	LastSeenAt time.Time
}

func (sessionModel) TableName() string {
	return "sessions"
}

func (m sessionModel) toDomain() *domain.Session {
	lastSeen := m.LastSeenAt
	if lastSeen.IsZero() {
		lastSeen = m.CreatedAt
	}

	return &domain.Session{
		ID:         m.ID,
		Token:      m.Token,
		UserID:     m.UserID,
		ExpiresAt:  m.ExpiresAt,
		LastSeenAt: lastSeen,
		CreatedAt:  m.CreatedAt,
	}
}

type UserStore struct {
	db *gorm.DB
}

func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Create(user *domain.User) error {
	role := user.Role
	if !role.Valid() {
		role = domain.RoleUser
	}

	model := &userModel{
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		AvatarPath:   user.AvatarPath,
		Role:         role,
	}

	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	user.ID = model.ID
	user.Role = model.Role
	user.CreatedAt = model.CreatedAt
	user.UpdatedAt = model.UpdatedAt
	return nil
}

func (s *UserStore) FindByID(id uint) (*domain.User, error) {
	var model userModel
	if err := s.db.First(&model, id).Error; err != nil {
		return nil, mapDBErr(err)
	}

	return model.toDomain(), nil
}

func (s *UserStore) FindByIDs(ids []uint) (map[uint]*domain.User, error) {
	out := make(map[uint]*domain.User)
	if len(ids) == 0 {
		return out, nil
	}

	var models []userModel
	if err := s.db.Where("id IN ?", ids).Find(&models).Error; err != nil {
		return nil, err
	}

	for i := range models {
		u := models[i].toDomain()
		out[u.ID] = u
	}

	return out, nil
}

func (s *UserStore) FindByUsername(username string) (*domain.User, error) {
	var model userModel
	if err := s.db.Where("username = ?", username).First(&model).Error; err != nil {
		return nil, mapDBErr(err)
	}

	return model.toDomain(), nil
}

func (s *UserStore) FindByEmail(email string) (*domain.User, error) {
	var model userModel
	if err := s.db.Where("email = ?", email).First(&model).Error; err != nil {
		return nil, mapDBErr(err)
	}

	return model.toDomain(), nil
}

func (s *UserStore) ExistsByUsername(username string) (bool, error) {
	var count int64
	err := s.db.Model(&userModel{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func (s *UserStore) ExistsByEmail(email string) (bool, error) {
	var count int64
	err := s.db.Model(&userModel{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (s *UserStore) Update(user *domain.User) error {
	role := user.Role
	if !role.Valid() {
		role = domain.RoleUser
	}

	return s.db.Model(&userModel{}).Where("id = ?", user.ID).Updates(map[string]any{
		"email":         user.Email,
		"password_hash": user.PasswordHash,
		"avatar_path":   user.AvatarPath,
		"role":          role,
	}).Error
}

func (s *UserStore) Delete(id uint) error {
	return s.db.Delete(&userModel{}, id).Error
}

func (s *UserStore) List(filter domain.UserListFilter) ([]domain.User, int, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset := max(filter.Offset, 0)

	q := s.db.Model(&userModel{})
	if query := strings.TrimSpace(filter.Query); query != "" {
		like := "%" + strings.ToLower(query) + "%"
		q = q.Where("lower(username) LIKE ? OR lower(email) LIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []userModel
	if err := q.Order("id ASC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	out := make([]domain.User, 0, len(models))
	for i := range models {
		out = append(out, *models[i].toDomain())
	}

	return out, int(total), nil
}

func (s *UserStore) CountByRole(role domain.Role) (int64, error) {
	if !role.Valid() {
		role = domain.RoleUser
	}

	var count int64
	err := s.db.Model(&userModel{}).Where("role = ?", role).Count(&count).Error
	return count, err
}

type SessionStore struct {
	db *gorm.DB
}

func NewSessionStore(db *gorm.DB) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) Create(session *domain.Session) error {
	lastSeen := session.LastSeenAt
	if lastSeen.IsZero() {
		lastSeen = time.Now().UTC()
	}

	model := &sessionModel{
		Token:      session.Token,
		UserID:     session.UserID,
		ExpiresAt:  session.ExpiresAt,
		LastSeenAt: lastSeen,
	}
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	session.ID = model.ID
	session.CreatedAt = model.CreatedAt
	session.LastSeenAt = model.LastSeenAt
	return nil
}

func (s *SessionStore) FindByToken(token string) (*domain.Session, error) {
	var model sessionModel
	if err := s.db.Where("token = ?", token).First(&model).Error; err != nil {
		return nil, mapDBErr(err)
	}
	return model.toDomain(), nil
}

func (s *SessionStore) Touch(session *domain.Session) error {
	return s.db.Model(&sessionModel{}).Where("id = ?", session.ID).Updates(map[string]any{
		"last_seen_at": session.LastSeenAt,
		"expires_at":   session.ExpiresAt,
	}).Error
}

func (s *SessionStore) DeleteByToken(token string) error {
	return s.db.Where("token = ?", token).Delete(&sessionModel{}).Error
}

func (s *SessionStore) DeleteByUserIDExcept(userID uint, keepToken string) error {
	if keepToken == "" {
		return s.DeleteByUserID(userID)
	}

	return s.db.Where("user_id = ? AND token <> ?", userID, keepToken).Delete(&sessionModel{}).Error
}

func (s *SessionStore) DeleteByUserID(userID uint) error {
	return s.db.Where("user_id = ?", userID).Delete(&sessionModel{}).Error
}
