# Repo

Лёгкий self-hosted Git-хостинг.

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

| Ключ                 | Описание                                                                             |
|----------------------|--------------------------------------------------------------------------------------|
| `http_addr`          | Адрес HTTP-сервера (по умолчанию `:8080`)                                            |
| `data_dir`           | Каталог для SQLite, Git-репозиториев и аватаров                                      |
| `site_name`          | Название сайта в UI                                                                  |
| `session_secret`     | Секрет подписи сессий (обязателен, мин. 32 символа; например `openssl rand -hex 32`) |
| `cookie_secure`      | `true` при работе за HTTPS                                                           |
| `allow_registration` | Публичная саморегистрация; иначе пользователи через CLI/админку                      |
| `receive_max_bytes`  | Максимальный размер Git push в байтах (по умолчанию 512 MiB)                         |
| `trusted_proxies`    | Опциональные CIDR/IP, которым доверять `X-Forwarded-For`                             |

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
