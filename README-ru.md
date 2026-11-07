# Repo

Лёгкий self-hosted Git-хостинг.

## Чем отличается

В отличие от GitHub, GitLab, Gitea и похожих хостингов, в Repo **нет организаций и групп** и **нет имени автора в URL репозитория**.

- Пути плоские: `https://example.com/my-app.git`, а не `https://example.com/utest/my-app.git`
- Имя репозитория уникально на всём инстансе (не в рамках пользователя или org)
- Доступ - через collaborators на репо и личные команды, а не через membership в org или вложенные группы

Рассчитан на небольшой self-hosted инстанс, где короткие clone URL важнее иерархии организаций.

## Возможности

- Репозитории по HTTP: clone, fetch, push; просмотр веток, тегов, дерева, коммитов, blame, README
- Issues и pull requests (diff, комментарии, merge)
- Плоские репозитории (`name.git`), collaborators на репозиторий и личные команды с правами read/write/admin на ваши репозитории
- Pull/push-зеркалирование по http(s) и SSH (интервал и окно синхронизации на репозиторий, статус и история, обзор для админа)
- Аккаунты, сессии, access-токены, админ-пользователи; UI на английском и русском

## Стек

- Go 1.27+, CGO для SQLite
- GORM + SQLite; go-git для операций с репозиториями
- Vue 3, Vite, TypeScript, Tailwind CSS, vue-i18n, vue-router
- Yarn для веб-приложения; Make для сборки

## Требования

- Go 1.27.1 или новее с включённым CGO (`gcc` / build essentials)
- Node.js 22.18+ или 24.12+, и Yarn
- `config.yaml` (скопируйте из `config.example.yaml`)

## Конфигурация

Скопируйте пример и отредактируйте значения:

```bash
cp config.example.yaml config.yaml
```

| Ключ                            | Описание                                                                                                     |
|---------------------------------|--------------------------------------------------------------------------------------------------------------|
| `http_addr`                     | Адрес HTTP-сервера (по умолчанию `:8080`)                                                                    |
| `data_dir`                      | Каталог для SQLite, Git-репозиториев и аватаров                                                              |
| `site_name`                     | Название сайта в UI                                                                                          |
| `session_secret`                | Секрет подписи сессий (обязателен, мин. 32 символа; например `openssl rand -hex 32`)                         |
| `cookie_secure`                 | `true` за HTTPS (Secure-cookie + HSTS)                                                                       |
| `allow_registration`            | Публичная саморегистрация (по умолчанию `true`); иначе пользователи через CLI/админку                        |
| `receive_max_bytes`             | Максимальный размер Git push в байтах (по умолчанию 512 MiB)                                                 |
| `mirror_pull_interval`          | Как часто pull-зеркала делают fetch (`5m` по умолчанию; `0` - без планировщика)                              |
| `mirror_verify_openpgp_keyring` | Armored OpenPGP public keyring для `mirror_verify_signed` (путь относительно `data_dir`, если не абсолютный) |
| `mirror_verify_allowed_signers` | Файл SSH `allowed_signers` (namespace `git`) для `mirror_verify_signed`                                      |
| `trusted_proxies`               | Опциональные CIDR/IP для `X-Forwarded-For` / `X-Forwarded-Proto`                                             |

В ответах выставляются CSP, `X-Frame-Options`, `X-Content-Type-Options` и (по HTTPS / при `cookie_secure`) HSTS с `includeSubDomains; preload`. После запуска сайта на HTTPS домен можно подать на [hstspreload.org](https://hstspreload.org).

## Зеркалирование

- **Pull**-зеркало синхронизируется с remote (здесь только чтение); **push**-зеркало публикуется после `git push` / merge
- Настройка при создании (*Зеркало* / *Импорт*) или в настройках репо; админ видит все зеркала в `/cp/mirrors`
- Опция `mirror_verify_signed`: после каждого pull tip должен проверяться по OpenPGP keyring и/или SSH `allowed_signers` сервера (не host GPG trustdb; рантайму не нужен системный бинарник `git`)
- Внутренние хосты по умолчанию заблокированы (SSRF); интервал и воркеры - в `config.example.yaml`

## Обратный прокси

Пример конфига nginx: `nginx.example.conf` (TLS, крупные Git push, прокси-заголовки). Укажите прокси в `trusted_proxies` и включите `cookie_secure: true`.

## Сборка

```bash
make install # yarn install в web/
make build # yarn build -> embed web/dist, затем go build -> build/repo-server
```

После `make build` бинарник - `./build/repo-server`. Релизный архив - `repo-linux-amd64.tar.gz`, внутри файл `repo-server`:

```bash
tar -xzf repo-linux-amd64.tar.gz
./repo-server serve
```

## Запуск

```bash
./build/repo-server serve
# или после распаковки релиза: ./repo-server serve
# или: make run
```

## Первый пользователь

Первого админа создают через CLI. Пароль - не короче 8 символов.

```bash
./build/repo-server user create -username admin -email admin@example.com -password 'your-password' -role admin
# бинарник из релиза: ./repo-server user create ...
```

## Тесты

```bash
make test
```
