# Changelog

## 0.1.0 - November 1, 2026

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
