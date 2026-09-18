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

cd web && yarn install && yarn dev
```

API reference: [API.md](API.md)

Locales: [langs/](langs/) (`en` / `ru`, `api.json` + `web.json`)
