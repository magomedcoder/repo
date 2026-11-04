# Changelog

# dev

- Sessions: store hashed tokens; list devices with IP and user agent; revoke one, other, or all sessions in settings
- Folders: shared org spaces with members (`owner` / `admin` / `member`), configurable capabilities, invite/edit/remove in the UI
- Teams: create teams inside a folder, add members, grant read/write/admin on selected repositories
- Collaborators: per-repository `admin` / `write` / `read`; Git HTTP and API use the same rules; folder membership does not imply repo write
- Git: enforce protected default branch on push after unpack; allow fast-forward, block force-push and delete; safer concurrent merge and push; gzip request bodies; private repos challenge with Basic auth
- Security: CSP and HSTS behind trusted HTTPS; CSRF bound to the session cookie; sanitize README HTML without rewriting images through a proxy
- Security: clamp folder member caps on update (no admin->owner / delete_folder escalation); revoke sessions and tokens on admin password or role change; uniform Git HTTP 401 for anonymous missing/private probes; block username/email reuse after soft-delete; require auth for collaborator list and visit beacons; escape admin LIKE wildcards; require password to change email
- Access: guests see only public activity, home, and search; sort repositories by name in the database; clamp list pagination
- Access tokens: optional expiry on create; expired tokens rejected for Git auth
- Issues and pull requests: monotonic number counters per repository
- Reliability: safer git repository open cache; create repository row before bare init; separate audit queues for security and visits; cancel browse work on request context

## v0.1.0 - November 1, 2026

- Accounts: register, login, logout, sessions, password change, registration toggle, login attempt lockouts
- Roles: user and admin; admin user list, create, update, delete in the API and UI
- Profile: public page, email update, avatar upload and delete
- Access tokens: create, list, and revoke in the API and UI for Git HTTP auth
- Folders: create, list, rename, delete; nested layout with ownerless repository paths
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
