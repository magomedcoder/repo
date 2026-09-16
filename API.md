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
curl -X POST http://127.0.0.1:8080/api/repos \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"my-repo","description":"My repository","folder_id":1,"private":false,"default_branch":"main"}'
```

## List repositories

```bash
# own repos (auth)
curl http://127.0.0.1:8080/api/repos -b cookies.txt

# public repos
curl "http://127.0.0.1:8080/api/repos?scope=public"
```

## Repository metadata

```bash
# root repo
curl http://127.0.0.1:8080/api/repos/user/hello

# nested repo
curl http://127.0.0.1:8080/api/repos/user/work/backend/api -b cookies.txt
```

## Update repository

```bash
curl -X PATCH http://127.0.0.1:8080/api/repos/user/work/backend/api \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"description":"Updated","private":true,"default_branch":"main"}'
```

## Move repository

```bash
curl -X POST http://127.0.0.1:8080/api/repos/user/work/backend/api/move \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"folder_id":null}'
```

## Delete repository

```bash
curl -X DELETE http://127.0.0.1:8080/api/repos/user/hello -b cookies.txt
```

## Browse repository

```bash
# branches / tags
curl "http://127.0.0.1:8080/api/repos/user/hello/branches"
curl "http://127.0.0.1:8080/api/repos/user/hello/tags"

# commits (paginated)
curl "http://127.0.0.1:8080/api/repos/user/hello/commits?ref=main&offset=0&limit=30"
curl "http://127.0.0.1:8080/api/repos/user/hello/commits/{sha}"
curl "http://127.0.0.1:8080/api/repos/user/hello/commits/{sha}/diff"

# tree / blob / raw
curl "http://127.0.0.1:8080/api/repos/user/hello/tree?ref=main&path="
curl "http://127.0.0.1:8080/api/repos/user/hello/blob?ref=main&path=README.md"
curl "http://127.0.0.1:8080/api/repos/user/hello/raw?ref=main&path=README.md"

# readme + stats
curl "http://127.0.0.1:8080/api/repos/user/hello/readme?ref=main"
curl "http://127.0.0.1:8080/api/repos/user/hello/stats?ref=main"
```

Nested repos use the same suffixes, e.g. `/api/repos/user/work/backend/api/tree?ref=main`.

## Personal access tokens

```bash
# create (token shown once)
curl -X POST http://127.0.0.1:8080/api/tokens \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"name":"cli"}'

# list
curl http://127.0.0.1:8080/api/tokens -b cookies.txt

# revoke
curl -X DELETE http://127.0.0.1:8080/api/tokens/1 -b cookies.txt
```

## Git clone / push (Smart HTTP)

```bash
# public clone
git clone http://127.0.0.1:8080/user/hello.git

# nested path
git clone http://127.0.0.1:8080/user/work/backend/api.git

# private / push - Basic auth with password or PAT
git clone http://user:PASSWORD@127.0.0.1:8080/user/private-repo.git

git clone http://user:repo_TOKEN@127.0.0.1:8080/user/private-repo.git

git push http://user:repo_TOKEN@127.0.0.1:8080/user/hello.git main
```

Force-push to the default branch is denied by a pre-receive hook.
