# Repo

Lightweight self-hosted Git hosting.

## How it differs

Unlike GitHub, GitLab, Gitea, and similar hosts, Repo has **no organizations or groups** and **no username in repository URLs**.

- Paths are flat: `https://example.com/my-app.git`, not `https://example.com/utest/my-app.git`
- Repository names are unique on the whole instance (not per user or org)

Built for a small self-hosted instance where short clone URLs matter more than multi-tenant org hierarchy.

## Features

- Repositories over HTTP: clone, fetch, push; browse branches, tags, tree, commits, blame, README
- Issues and pull requests (diff, comments, merge); optional default-branch protection (blocks force-push/delete for all heads and tags while enabled)
- Flat repositories (`<name>.git`), per-repo collaborators, and personal teams that grant read/write/admin on your repositories
- Pull and push mirroring over http(s) and SSH (interval, sync window, status/history, admin overview)
- Accounts, sessions, access tokens (`git:read` / `git:write`), admin panel at `/cp`; UI in English and Russian
- Code and repository search; activity feed

## Build and run

```bash
git clone https://magomedcoder.ru/repo.git
cd repo

cp config.example.yaml config.yaml
chmod 600 config.yaml
# fill secrets.session_secret, token_pepper, csrf_secret, mirror_secrets_key
# each: openssl rand -hex 32

make install   # yarn install in web/
make build     # yarn build -> embed web/dist, then go build -> build/repo-server

# first admin (password ≥ 8 characters)
./build/repo-server user create -username admin -email admin@example.com -password 'your-password' -role admin

./build/repo-server serve
# or: make run
```

Open `http://127.0.0.1:8080`. Admin UI: `/cp`.

Production tip: reverse proxy + `http.cookie_secure: true` + prefer `auth.git_password_auth: false` (PATs only).

## Requirements

- Go 1.27.3+ with CGO (`gcc` / build essentials); Node.js 22.18+ or 24.12+, Yarn
- Stack: GORM + SQLite, go-git, Vue 3 / Vite / TypeScript / Tailwind; `make` orchestrates the build
- `config.yaml` (copy from `config.example.yaml`; mode `0600`/`0400` or the server refuses to start)

## Configuration

Copy the example and edit values:

```bash
cp config.example.yaml config.yaml
```

Generate each secret with `openssl rand -hex 32`. All four secrets are required (min 32 characters each) and must not fall back onto each other. Nested YAML only (`http` / `server` / `secrets` / `auth` / `mirror`); flat keys are rejected.

```yaml
http:
  # Bind loopback behind a reverse proxy. Non-loopback requires cookie_secure: true.
  addr: "127.0.0.1:8080"
  # true (HTTPS / reverse proxy) -> __Host-session / __Host-repo_csrf (Secure).
  # false (loopback/dev only) -> plain session / repo_csrf; __Host- is never issued without Secure.
  cookie_secure: true
  # Real proxy CIDRs for X-Forwarded-For / Proto. Never 0.0.0.0/0 or ::/0. Empty if no proxy.
  trusted_proxies:
    - "127.0.0.1/32"
    - "::1/128"
  receive_max_bytes: 268435456   # max Git push size (default 256 MiB)

server:
  data_dir: "data"               # SQLite, Git repos, avatars
  site_name: "Repo"              # UI title
  soft_deleted_retention: 168h   # purge soft-deleted repos after this

secrets:
  session_secret: ""             # openssl rand -hex 32 - signs session cookies
  token_pepper: ""               # hashes personal access tokens
  csrf_secret: ""                # binds CSRF tokens to the session
  mirror_secrets_key: ""         # encrypts stored mirror credentials

auth:
  allow_registration: false      # public self-signup; keep false on public instances
  allow_public_contribute: false # any logged-in user may open issues/PRs on public repos
  require_auth_for_read: false   # true = login required for all reads (incl. public)
  git_password_auth: true        # false = Git HTTP accepts only PATs; prefer false in prod (password ignores PAT repo_ids)
  session_absolute: 720h         # hard session lifetime
  session_idle: 336h             # idle timeout (refreshed on activity)
  # Login / Git password soft-lock is in-memory only: process restart clears lockouts.

mirror:
  pull_interval: 5m              # how often pull mirrors fetch; 0 disables scheduler
  workers: 4                     # concurrent mirror workers
  clone_timeout: 15m
  fetch_timeout: 10m
  push_timeout: 10m
  default_max_bytes: 5368709120  # used when a mirror leaves max_bytes at 0 (5 GiB)
  # verify:                      # relative to data_dir only (absolute paths rejected at startup)
  #   openpgp_keyring: "trust/openpgp.asc"
  #   allowed_signers: "trust/allowed_signers"
```

Git **push** always challenges with `401` Basic. Write needs owner or collaborator/team with write. Anonymous **read** of private repos returns `404`. Auth: account password (if `auth.git_password_auth: true`) or a `git:write` PAT.

Responses set CSP and related headers; with HTTPS / `http.cookie_secure`, HSTS is enabled. Failed logins share an in-memory soft-lock (`429` + `Retry-After`); a process restart clears it.

## Mirroring

Pull (read-only here) and push after local `git push` / merge. Configure in the UI or `/cp/mirrors`. Tip signature checks use `mirror.verify` (files under `data_dir`). Internal hosts are blocked (SSRF); changing URL/secrets needs `current_password`.

## Reverse proxy

[`nginx.example.conf`](./nginx.example.conf): TLS and Git pushes. Set `http.trusted_proxies` and `http.cookie_secure: true`.

## Test

```bash
make test
```
