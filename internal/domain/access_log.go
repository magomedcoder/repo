package domain

import "time"

const (
	AccessVisit           = "visit"
	AccessClone           = "clone"
	AccessDownload        = "download"
	AccessPush            = "push"
	AccessLogin           = "login"
	AccessLogout          = "logout"
	AccessRegister        = "register"
	AccessFailedAuth      = "failed_auth"
	AccessTokenCreate     = "token_create"
	AccessTokenRevoke     = "token_revoke"
	AccessRepoCreate      = "repo_create"
	AccessRepoDelete      = "repo_delete"
	AccessRepoMove        = "repo_move"
	AccessAdminUserCreate = "admin_user_create"
	AccessAdminUserUpdate = "admin_user_update"
	AccessAdminUserDelete = "admin_user_delete"
)

type AccessLog struct {
	ID        uint
	Action    string
	IP        string
	UserAgent string
	UserID    *uint
	Username  string
	Path      string
	Repo      string
	Ref       string
	CreatedAt time.Time
}

type AccessLogFilter struct {
	Action   string
	IP       string
	Username string
	Limit    int
	Offset   int
}

type AccessLogStore interface {
	Create(log *AccessLog) error

	List(filter AccessLogFilter) (items []AccessLog, total int, err error)
}
