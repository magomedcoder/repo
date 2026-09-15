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

## Folders

```bash
# create root folder
curl -X POST http://127.0.0.1:8080/api/folders \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"work"}'

# create nested folder by parent_id
curl -X POST http://127.0.0.1:8080/api/folders \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"backend","parent_id":1}'

# or by parent path
curl -X POST http://127.0.0.1:8080/api/folders \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"frontend","path":"work"}'

# list children (root by default)
curl "http://127.0.0.1:8080/api/folders" -b cookies.txt

# full tree
curl "http://127.0.0.1:8080/api/folders?tree=1" -b cookies.txt

# folder contents (subfolders + repos)
curl http://127.0.0.1:8080/api/folders/1 -b cookies.txt

# rename
curl -X PATCH http://127.0.0.1:8080/api/folders/2 \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"api"}'

# move to another parent (null = root)
curl -X POST http://127.0.0.1:8080/api/folders/2/move \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"parent_id":1}'

# delete (only if empty)
curl -X DELETE http://127.0.0.1:8080/api/folders/2 -b cookies.txt
```

## Create repository

```bash
curl -X POST http://127.0.0.1:8080/api/user/repos \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"my-repo","description":"My repository","folder_id":1}'
```

## List repositories

```bash
curl http://127.0.0.1:8080/api/user/repos -b cookies.txt
```
