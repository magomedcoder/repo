# Repo

Lightweight self-hosted Git hosting.

## How it differs

Unlike GitHub, GitLab, Gitea, and similar hosts, Repo has **no organizations or groups** and **no username in repository URLs**.

- Paths are flat: `https://example.com/my-app.git`, not `https://example.com/utest/my-app.git`
- Repository names are unique on the whole instance (not per user or org)
- Access is via per-repo collaborators and personal teams - not org membership or nested group trees

Built for a small self-hosted instance where short clone URLs matter more than multi-tenant org hierarchy.

## Features

- Repositories over HTTP: clone, fetch, push; browse branches, tags, tree, commits, blame, README
- Issues and pull requests (diff, comments, merge)
- Flat repositories (`name.git`), per-repo collaborators, and personal teams that grant read/write/admin on your repositories
- Pull and push mirroring over http(s) and SSH (per-repository interval and sync window, status and history, admin overview)
- Accounts, sessions, access tokens, admin users; UI in English and Russian

## Stack

- Go 1.27+, CGO for SQLite
- GORM + SQLite; go-git for repository operations
- Vue 3, Vite, TypeScript, Tailwind CSS, vue-i18n, vue-router
- Yarn for the web app; Make for build orchestration

## Requirements

- Go 1.27.1 or newer with CGO enabled (`gcc` / build essentials)
- Node.js 22.18+ or 24.12+, and Yarn
- `config.yaml` (copy from `config.example.yaml`)

## Configuration

Copy the example and edit values:

```bash
cp config.example.yaml config.yaml
```

| Key                             | Description                                                                                       |
|---------------------------------|---------------------------------------------------------------------------------------------------|
| `http_addr`                     | HTTP listen address (default `:8080`)                                                             |
| `data_dir`                      | SQLite DB, Git repos, and avatars                                                                 |
| `site_name`                     | Site title in the UI                                                                              |
| `session_secret`                | Session signing secret (required, min 32 characters; e.g. `openssl rand -hex 32`)                 |
| `cookie_secure`                 | Set `true` behind HTTPS (Secure cookies + HSTS)                                                   |
| `allow_registration`            | Public self-registration (default `true`); otherwise create users via CLI/admin                   |
| `receive_max_bytes`             | Max Git push size in bytes (default 512 MiB)                                                      |
| `mirror_pull_interval`          | How often pull mirrors fetch (`5m` by default; `0` disables the scheduler)                        |
| `mirror_verify_openpgp_keyring` | Armored OpenPGP public keyring for `mirror_verify_signed` (path under `data_dir` unless absolute) |
| `mirror_verify_allowed_signers` | SSH `allowed_signers` file (namespace `git`) for `mirror_verify_signed`                           |
| `trusted_proxies`               | Optional CIDRs/IPs trusted for `X-Forwarded-For` / `X-Forwarded-Proto`                            |

Responses include CSP, `X-Frame-Options`, `X-Content-Type-Options`, and (over HTTPS / with `cookie_secure`) HSTS with `includeSubDomains; preload`. After the site is live on HTTPS, you can submit the domain at [hstspreload.org](https://hstspreload.org).

## Mirroring

- **Pull** mirrors stay in sync with a remote (read-only here); **push** mirrors publish after `git push` / merge
- Configure on create (*Mirror* / *Import*) or in repository settings; admins manage all mirrors at `/cp/mirrors`
- Optional `mirror_verify_signed`: after each pull, the tip must verify against the server OpenPGP keyring and/or SSH `allowed_signers` (not the host GPG trustdb; runtime does not need a system `git` binary)
- Internal hosts blocked by default (SSRF); interval and workers in `config.example.yaml`

## Reverse proxy

Example nginx config: `nginx.example.conf` (TLS termination, large Git pushes, forwarded headers). Point `trusted_proxies` at the proxy and set `cookie_secure: true`.

## Build

```bash
make install # yarn install in web/
make build # yarn build -> embed web/dist, then go build -> build/repo-server
```

After `make build` the binary is `./build/repo-server`. A release asset is `repo-linux-amd64.tar.gz` with `repo-server` inside:

```bash
tar -xzf repo-linux-amd64.tar.gz
./repo-server serve
```

## Run

```bash
./build/repo-server serve
# or after unpacking a release: ./repo-server serve
# or: make run
```

## First user

Create the first admin via CLI. Password must be at least 8 characters.

```bash
./build/repo-server user create -username admin -email admin@example.com -password 'your-password' -role admin
# release binary: ./repo-server user create ...
```

## Test

```bash
make test
```
