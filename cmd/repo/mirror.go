package main

import (
	"gorm.io/gorm"

	"github.com/magomedcoder/repo/internal/adapter/sqlite"
	"github.com/magomedcoder/repo/internal/config"
	"github.com/magomedcoder/repo/internal/usecase"
)

func setupMirror(uc *usecase.RepositoryUseCase, cfg config.Config, db *gorm.DB) error {
	uc.SetMirrorConfig(usecase.MirrorConfigFromApp(cfg))
	uc.SetMirrorSyncStore(sqlite.NewMirrorSyncRepo(db))
	uc.SetMirrorGroupStore(sqlite.NewMirrorGroupRepo(db))

	return nil
}
