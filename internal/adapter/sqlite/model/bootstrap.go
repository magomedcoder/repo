package model

func BootstrapModels() []any {
	return []any{
		&User{},
		&Session{},
		&AccessToken{},
		&Folder{},
		&Repository{},
		&FolderMember{},
		&Team{},
		&TeamMember{},
		&TeamRepoGrant{},
		&RepoCollaborator{},
		&Issue{},
		&IssueComment{},
		&Label{},
		&IssueLabel{},
		&PullRequest{},
		&PullComment{},
		&Activity{},
		&AccessLog{},
		&MirrorSyncEvent{},
		&MirrorTarget{},
		&MirrorGroup{},
	}
}
