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

## Issues

```bash
# list (state=open|closed|all, default open)
curl "http://127.0.0.1:8080/api/repos/user/hello/issues?state=open&offset=0&limit=30"

# create
curl -X POST http://127.0.0.1:8080/api/repos/user/hello/issues \
  -H "Content-Type: application/json" -b cookies.txt \
  -d '{"title":"Bug","body":"steps","label_ids":[1]}'

# get
curl http://127.0.0.1:8080/api/repos/user/hello/issues/1

# update (title, body, state, label_ids are optional)
curl -X PATCH http://127.0.0.1:8080/api/repos/user/hello/issues/1 \
  -H "Content-Type: application/json" -b cookies.txt \
  -d '{"state":"closed"}'

# delete
curl -X DELETE http://127.0.0.1:8080/api/repos/user/hello/issues/1 -b cookies.txt

# comments
curl -X POST http://127.0.0.1:8080/api/repos/user/hello/issues/1/comments \
  -H "Content-Type: application/json" -b cookies.txt \
  -d '{"body":"looks good"}'
curl -X PATCH http://127.0.0.1:8080/api/repos/user/hello/issues/1/comments/1 \
  -H "Content-Type: application/json" -b cookies.txt \
  -d '{"body":"updated"}'
curl -X DELETE http://127.0.0.1:8080/api/repos/user/hello/issues/1/comments/1 -b cookies.txt

# labels (list is readable with the repo; create/update/delete are owner-only)
curl http://127.0.0.1:8080/api/repos/user/hello/labels
curl -X POST http://127.0.0.1:8080/api/repos/user/hello/labels \
  -H "Content-Type: application/json" -b cookies.txt \
  -d '{"name":"bug","color":"#1f6b4f"}'
curl -X PATCH http://127.0.0.1:8080/api/repos/user/hello/labels/1 \
  -H "Content-Type: application/json" -b cookies.txt \
  -d '{"name":"bugfix"}'
curl -X DELETE http://127.0.0.1:8080/api/repos/user/hello/labels/1 -b cookies.txt
```

## Pull requests

Same-repository branch compare only (no forks). Create/comment: any logged-in user who can read the repo. Merge and base/head changes: owner only. Close/reopen/delete: author or owner. States: `open`, `closed`, `merged`. Merge strategies: `merge` (default) or `ff-only`.

```bash
# list (state=open|closed|merged|all)
curl "http://127.0.0.1:8080/api/repos/user/hello/pulls?state=open"

# create
curl -X POST http://127.0.0.1:8080/api/repos/user/hello/pulls \
  -H "Content-Type: application/json" -b cookies.txt \
  -d '{"title":"Add feature","body":"details","base_branch":"main","head_branch":"feature"}'

# get / diff / commits
curl http://127.0.0.1:8080/api/repos/user/hello/pulls/1
curl http://127.0.0.1:8080/api/repos/user/hello/pulls/1/diff
curl http://127.0.0.1:8080/api/repos/user/hello/pulls/1/commits

# update / merge / delete
curl -X PATCH http://127.0.0.1:8080/api/repos/user/hello/pulls/1 \
  -H "Content-Type: application/json" -b cookies.txt \
  -d '{"state":"closed"}'
curl -X POST http://127.0.0.1:8080/api/repos/user/hello/pulls/1/merge \
  -H "Content-Type: application/json" -b cookies.txt \
  -d '{"strategy":"merge"}'
curl -X DELETE http://127.0.0.1:8080/api/repos/user/hello/pulls/1 -b cookies.txt

# comments
curl -X POST http://127.0.0.1:8080/api/repos/user/hello/pulls/1/comments \
  -H "Content-Type: application/json" -b cookies.txt \
  -d '{"body":"lgtm"}'
```

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

## SSH keys

```bash
# add
curl -X POST http://127.0.0.1:8080/api/ssh-keys \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{"title":"laptop","public_key":"ssh-ed25519 AAAA... comment"}'

# list
curl http://127.0.0.1:8080/api/ssh-keys -b cookies.txt

# delete
curl -X DELETE http://127.0.0.1:8080/api/ssh-keys/1 -b cookies.txt
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

## Git clone / push (SSH)

The app listens for SSH on `:2222` by default (`REPO_SSH_ADDR`). Host keys are stored in `data/ssh/`. Auth is by registered public key; the SSH username is ignored.

```bash
# custom port
git clone ssh://git@127.0.0.1:2222/user/hello.git
git clone ssh://git@127.0.0.1:2222/user/work/backend/api.git

# if REPO_SSH_ADDR=:22
git clone git@host:user/hello.git
```

Force-push to the default branch is denied by a pre-receive hook.
