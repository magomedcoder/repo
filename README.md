# Repo

Lightweight self-hosted Git hosting.

Built with go-git, SQLite, GORM.

Most Git hostings group projects with organizations and teams.
Repo skips that model: each user has a personal namespace, and repositories live in nested folders - like a filesystem.

```
user/
├── work/
│   ├── backend/api.git
│   └── frontend/web.git
└── playground/hello.git
```

## Run

```bash
go run ./cmd/repo
```

# API

## Create repository

```bash
curl -X POST http://127.0.0.1:8080/api/user/repos \
  -H "Content-Type: application/json" \
  -d '{"name":"my-repo","description":"My repository"}'
```
