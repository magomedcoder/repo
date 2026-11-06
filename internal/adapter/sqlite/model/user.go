package model

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type User struct {
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

func (User) TableName() string {
	return "users"
}

func (m User) ToDomain() *domain.User {
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
