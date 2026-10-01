# Repo

[English version](README.md)

Лёгкий self-hosted Git-хостинг.

Стек: go-git, SQLite, GORM.

Большинство Git-хостингов группируют проекты через организации и команды.
Repo этого не делает: у каждого пользователя своё пространство, а репозитории лежат во вложенных папках - как в файловой системе.

```
user/
├── work/
│   ├── backend/api.git
│   └── frontend/web.git
└── playground/hello.git
```

## Запуск

```bash
go run ./cmd/repo

cd web && yarn install && yarn dev
```

HTTP API слушает `:8080`. SSH для git - `:2222` (`REPO_SSH_ADDR`).

Справка по API: [API.md](API.md)

Локализация: [langs/](langs/) (`en` / `ru`, `api.json` + `web.json`)
