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

## Register

```bash
curl -X POST http://127.0.0.1:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -c cookies.txt \
  -d '{"username":"user","email":"user@example.com","password":"secret123"}'
```

## Login

```bash
curl -X POST http://127.0.0.1:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -c cookies.txt \
  -d '{"login":"user","password":"secret123"}'
```

## Me

```bash
curl http://127.0.0.1:8080/api/auth/me -b cookies.txt
```

## Logout

```bash
curl -X POST http://127.0.0.1:8080/api/auth/logout -b cookies.txt -c cookies.txt
```

## Create repository

```bash
curl -X POST http://127.0.0.1:8080/api/user/repos \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"my-repo","description":"My repository"}'
```
