# Repo

Лёгкий self-hosted Git-хостинг.

## Чем отличается

В отличие от GitHub, GitLab, Gitea и похожих хостингов, в Repo **нет организаций и групп** и **нет имени автора в URL репозитория**.

- Пути плоские: `https://example.com/my-app.git`, а не `https://example.com/utest/my-app.git`
- Имя репозитория уникально на всём инстансе (не в рамках пользователя или org)

Рассчитан на небольшой self-hosted инстанс, где короткие clone URL важнее иерархии организаций.

## Возможности

- Репозитории по HTTP: clone, fetch, push; просмотр веток, тегов, дерева, коммитов, blame, README
- Issues и pull requests (diff, комментарии, merge); опциональная защита ветки по умолчанию (пока включена - запрет force-push/delete для всех heads и tags)
- Плоские репозитории (`<name>.git`), collaborators на репозиторий и личные команды с правами read/write/admin
- Pull/push-зеркалирование по http(s) и SSH (интервал, окно синхронизации, статус/история, обзор для админа)
- Аккаунты, сессии, access-токены (`git:read` / `git:write`), панель `/cp`; UI на английском и русском
- Поиск по коду и репозиториям; лента активности

## Сборка и запуск

```bash
git clone https://github.com/magomedcoder/repo.git
cd repo

cp config.example.yaml config.yaml
chmod 600 config.yaml
# заполните secrets.session_secret, token_pepper, csrf_secret, mirror_secrets_key
# каждый: openssl rand -hex 32

make install   # yarn install в web/
make build     # yarn build -> embed web/dist, затем go build -> build/repo-server

# первый админ (пароль ≥ 8 символов)
./build/repo-server user create -username admin -email admin@example.com -password 'your-password' -role admin

./build/repo-server serve
# или: make run
```

Откройте `http://127.0.0.1:8080`. Админка: `/cp`.

Для продакшена: reverse proxy + `http.cookie_secure: true` + лучше `auth.git_password_auth: false` (только PAT).

## Требования

- Go 1.27.3+ с CGO (`gcc` / build essentials); Node.js 22.18+ или 24.12+, Yarn
- Стек: GORM + SQLite, go-git, Vue 3 / Vite / TypeScript / Tailwind; сборку ведёт `make`
- `config.yaml` (из `config.example.yaml`; права `0600`/`0400`, иначе сервер не стартует)

## Конфигурация

Скопируйте пример и отредактируйте значения:

```bash
cp config.example.yaml config.yaml
```

Каждый секрет генерируйте через `openssl rand -hex 32`. Все четыре секрета обязательны (мин. 32 символа каждый) и не подставляются друг за друга. Только вложенный YAML (`http` / `server` / `secrets` / `auth` / `mirror`); плоские ключи отклоняются.

```yaml
http:
  # Loopback за reverse proxy. Не-loopback требует cookie_secure: true.
  addr: "127.0.0.1:8080"
  # true (HTTPS / reverse proxy) -> __Host-session / __Host-repo_csrf (Secure).
  # false (только loopback/dev) -> обычные session / repo_csrf; __Host- без Secure не выдаётся.
  cookie_secure: true
  # Реальные CIDR прокси для X-Forwarded-For / Proto. Нельзя 0.0.0.0/0 и ::/0. Пусто - если прокси нет.
  trusted_proxies:
    - "127.0.0.1/32"
    - "::1/128"
  receive_max_bytes: 268435456   # макс. размер Git push (по умолчанию 256 MiB)

server:
  data_dir: "data"               # SQLite, Git-репо, аватары
  site_name: "Repo"              # название в UI
  soft_deleted_retention: 168h   # через сколько чистить мягко удалённые репо

secrets:
  session_secret: ""             # openssl rand -hex 32 - подпись session-cookie
  token_pepper: ""               # хеширование personal access tokens
  csrf_secret: ""                # привязка CSRF к сессии
  mirror_secrets_key: ""         # шифрование учётных данных зеркал

auth:
  allow_registration: false      # публичная саморегистрация; на публичных инстансах оставляйте false
  allow_public_contribute: false # любой залогиненный может открывать issues/PR на public-репо
  require_auth_for_read: false   # true = вход обязателен для любого чтения (в т.ч. public)
  git_password_auth: true        # false = на Git HTTP только PAT; в проде лучше false (пароль игнорирует PAT repo_ids)
  session_absolute: 720h         # жёсткий срок жизни сессии
  session_idle: 336h             # idle-timeout (обновляется при активности)
  # Soft-lock логина / Git password только in-memory: рестарт процесса сбрасывает блокировки.

mirror:
  pull_interval: 5m              # как часто pull-зеркала делают fetch; 0 - без планировщика
  workers: 4                     # параллельные воркеры зеркалирования
  clone_timeout: 15m
  fetch_timeout: 10m
  push_timeout: 10m
  default_max_bytes: 5368709120  # если у зеркала max_bytes = 0 (5 ГиБ)
  # verify:                      # только относительно data_dir (абсолютные пути отклоняются)
  #   openpgp_keyring: "trust/openpgp.asc"
  #   allowed_signers: "trust/allowed_signers"
```

**Push** всегда отвечает `401` Basic. Write - только owner или collaborator/team с write. Анонимное **чтение** private даёт `404`. Авторизация: пароль аккаунта (если `auth.git_password_auth: true`) или PAT с `git:write`.

В ответах - CSP и смежные заголовки; при HTTPS / `http.cookie_secure` включается HSTS. Неудачные логины делят in-memory soft-lock (`429` + `Retry-After`); рестарт процесса сбрасывает блокировку.

## Зеркалирование

Pull (только чтение здесь) и push после локального `git push` / merge. Настройка в UI или `/cp/mirrors`. Проверка подписи tip - через `mirror.verify` (файлы под `data_dir`). Внутренние хосты режутся (SSRF); смена URL/секретов - с `current_password`.

## Обратный прокси

[`nginx.example.conf`](./nginx.example.conf): TLS и Git push. Задайте `http.trusted_proxies` и `http.cookie_secure: true`.

## Тесты

```bash
make test
```
