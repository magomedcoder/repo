package model

func BootstrapModels() []any {
	return []any{
		&User{},
		&Session{},
		&AccessToken{},
		&Repository{},
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
		&MirrorGroup{},
	}
}
