# Repo

Lightweight self-hosted Git hosting.

## Features

- Repositories over HTTP: clone, fetch, push; browse branches, tags, tree, commits, blame, README
- Issues and pull requests (diff, comments, merge)
- Folders as shared org spaces: members, capabilities, teams, per-repo collaborators
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

| Key                    | Description                                                                       |
|------------------------|-----------------------------------------------------------------------------------|
| `http_addr`            | HTTP listen address (default `:8080`)                                             |
| `data_dir`             | SQLite DB, Git repos, and avatars                                                 |
| `site_name`            | Site title in the UI                                                              |
| `session_secret`       | Session signing secret (required, min 32 characters; e.g. `openssl rand -hex 32`) |
| `cookie_secure`        | Set `true` behind HTTPS (Secure cookies + HSTS)                                   |
| `allow_registration`   | Public self-registration; otherwise create users via CLI/admin                    |
| `receive_max_bytes`    | Max Git push size in bytes (default 512 MiB)                                      |
| `mirror_pull_interval` | How often pull mirrors fetch (`5m` by default; `0` disables the scheduler)        |
| `trusted_proxies`      | Optional CIDRs/IPs trusted for `X-Forwarded-For` / `X-Forwarded-Proto`            |

Responses include CSP, `X-Frame-Options`, `X-Content-Type-Options`, and (over HTTPS / with `cookie_secure`) HSTS with `includeSubDomains; preload`. After the site is live on HTTPS, you can submit the domain at [hstspreload.org](https://hstspreload.org).

## Mirroring

Any repository can be a **pull mirror** (kept in sync with a remote, read-only here) or a **push mirror** (published to a remote after `git push` / merge). Set it up when creating a repository (*Mirror* or one-time *Import*) or under repository **Settings -> Mirroring**: interval, sync window, force/prune, refspecs, LFS, size limit, credentials or SSH key, *Check connection*, sync history and a next-pull preview. Pull mirrors show a read-only banner with an upstream link and a syncing / ok / error / paused indicator; instance admins get all mirrors with bulk sync and pause under **Mirrors** (`/settings/mirrors`).

Internal hosts are blocked by default (SSRF protection). Interval, workers, and timeouts are set in `config.example.yaml`.

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
