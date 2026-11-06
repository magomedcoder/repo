package main

import (
	"gorm.io/gorm"

	"magomedcoder.ru/repo/internal/adapter/sqlite"
	"magomedcoder.ru/repo/internal/config"
	"magomedcoder.ru/repo/internal/usecase"
)

func setupMirror(uc *usecase.RepositoryUseCase, cfg config.Config, db *gorm.DB) error {
	uc.SetMirrorConfig(usecase.MirrorConfigFromApp(cfg))
	uc.SetMirrorSyncStore(sqlite.NewMirrorSyncRepo(db))
	uc.SetMirrorTargetStore(sqlite.NewMirrorTargetRepo(db))
	uc.SetMirrorGroupStore(sqlite.NewMirrorGroupRepo(db))

	return nil
}
