# Changelog

## dev

- Updated time on home and lists: from `last_activity_at` only (git push and merge); mirror sync and settings no longer bump it
- Admin repositories: sync Updated from the tip commit on each default branch
- Default branch change: set Updated from the new branch tip commit
- SQLite: `schema_migrations` journal; pending steps only; schema verify after bootstrap or new migrations
- SEO: `public_url`, document meta injection, robots.txt, sitemap.xml

## v1.0.0 - November 8, 2026

- Accounts: register, login, logout, sessions, password change, registration toggle, login attempt lockouts
- Roles: user and admin; admin user list, create, update, delete in the API and UI; control panel at `/cp`
- Profile: public page, email update, avatar upload and delete
- Access tokens: create, list, and revoke for Git HTTP; default 90-day expiry; `git:read` / `git:write` scopes; HMAC pepper
- Sessions: hashed tokens; list devices with IP and user agent; revoke one, others, or all
- Repositories: flat paths (`name.git`); create, update, rename, delete; description; public and private; default branch; protect branches; issues and pull requests toggles; show on home
- Collaborators: per-repository `admin` / `write` / `read`; same rules for Git HTTP and API
- Teams: personal user-owned teams under `/settings/teams` and `/api/teams`; members and repo grants
- Access: guest read for public repos; write requires login; guests see only public activity, home, and search; private Git HTTP returns 404 for outsiders
- Git over HTTP: clone, fetch, push; fast-forward-only branch protection; Basic auth for private repos
- Browse: branches, tags, tree, blob, raw download, commits, commit diff, blame, summary (readme and stats)
- Issues: open, close, edit, delete, comments, labels; monotonic numbers per repository
- Pull requests: open, close, edit, delete, comments, diff, commits, merge
- Mirroring: pull and push over http(s) and SSH; schedule, pause, manual sync; Import vs Mirror create; admin Mirrors page with bulk sync, groups, and verify
- Search repositories by name; public home listing
- Activity feed for push, issues, pull requests, and repository creation
- YAML configuration (nested `http` / `server` / `secrets` / `auth` / `mirror`) for listen address, storage, site name, cookies, sessions, registration, proxies, receive limits, and mirror options
- Git HTTP push: anonymous receive-pack returns `401` Basic challenge (including private repos); write requires owner or collaborator/team with write
- UI in English and Russian
- Security: rate limiting, CSRF, CSP (+ CSP-report IP limit), HSTS; Argon2id; step-up password for sensitive admin, delete, and revoke-other-sessions; admin user-delete holds repo names; `git:write` PAT needs write ACL; mirror `insecure_tls` requires host pins and same-host redirects; trust paths under `data_dir`; default receive 256 MiB / mirror max 5 GiB; configurable session TTL
- Maintenance: prune expired sessions/tokens, old activity, and soft-deleted repositories; SQLite schema bootstrap on empty database; CLI to serve the app and create users
