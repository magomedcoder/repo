# Changelog

# dev

- Mirroring: pull and push from http(s) remotes on create and in settings; scheduled pull sync, push sync after git push or merge; manual sync; pull mirrors reject git push
- Mirroring options: pause without removing; interval, sync window, force/prune, refspecs, LFS, size limit, conflict policy; SSH key with passphrase; insecure TLS for admins
- Mirroring UI: connection check, sync history, next-pull preview; status on pull mirrors; Import vs Mirror create flow; admin Mirrors page with bulk sync and pause
- Mirror integrity: verify remote refs after sync and on demand from the repository or admin API
- Sessions: hashed tokens; list devices with IP and user agent; revoke one, others, or all in settings
- Folders: shared spaces with members (`owner` / `admin` / `member`), capabilities, invite/edit/remove in the UI
- Teams: create teams in a folder, add members, grant read/write/admin on selected repositories
- Collaborators: per-repository `admin` / `write` / `read`; same rules for Git HTTP and API; folder membership does not grant repo write
- Access tokens: optional expiry on create; expired tokens rejected for Git auth
- Issues and pull requests: monotonic number counters per repository
- Access: guests see only public activity, home, and search; repository lists sorted by name with clamped pagination
- Git: protected default branch after unpack (fast-forward only; block force-push and delete); Basic auth challenge for private repos; gzip request bodies
- Security: CSP and HSTS behind trusted HTTPS; session-bound CSRF; safer README HTML; password required to change email
- Security: clamp folder member caps; revoke sessions and tokens when an admin changes password or role; block username/email reuse after soft-delete; auth required for collaborator list and visit beacons
- Maintenance: hourly prune of expired sessions/tokens, old activity, and soft-deleted repositories; WAL checkpoint; tighter rate-limit and cache cleanup
- Reliability: schema verify after migrations; safer git open cache and repository create path; cancel browse work when the request ends

## v0.1.0 - November 1, 2026

- Accounts: register, login, logout, sessions, password change, registration toggle, login attempt lockouts
- Roles: user and admin; admin user list, create, update, delete in the API and UI
- Profile: public page, email update, avatar upload and delete
- Access tokens: create, list, and revoke in the API and UI for Git HTTP auth
- Folders: create, list, rename, delete; flat layout with ownerless repository paths
- Repositories: create, update, rename/move, delete, description, public and private, default branch, protect default branch, issues and pull requests toggles
- Access: guest read for public repos; write actions require login; settings only for owners
- Git over HTTP: clone, fetch, push; force-push guard on protected default branch
- Browse: branches, tags, tree, blob, raw download, commits, commit diff, blame, readme, summary, stats
- Issues: open, close, edit, delete, comments, labels
- Pull requests: open, close, edit, delete, comments, diff, commits, merge
- Search repositories by name; public home listing
- Activity feed for push, issues, pull requests, and repository creation
- YAML configuration for server, storage, site name, cookies, sessions, registration, proxies, and receive limits
- UI in English and Russian
- Rate limiting, CSRF, security headers; SQLite schema bootstrap and versioned migrations
- CLI to serve the app and create users
