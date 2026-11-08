# Changelog

## v1.0.0 - 8 ноября 2026

- Аккаунты: регистрация, вход, выход, сессии, смена пароля, переключатель регистрации, блокировка после неудачных попыток входа
- Роли: user и admin; список, создание, изменение и удаление пользователей для админа в API и UI; панель `/cp`
- Профиль: публичная страница, смена email, загрузка и удаление аватара
- Access-токены: создание, список и отзыв для Git HTTP; по умолчанию 90 дней; scopes `git:read` / `git:write`; HMAC pepper
- Сессии: хешированные токены; список устройств с IP и user agent; отзыв одной, остальных или всех
- Репозитории: плоские пути (`name.git`); создание, обновление, rename, удаление; описание; public и private; default branch; защита веток; переключатели issues и pull requests; показ на главной
- Collaborators: на репозиторий `admin` / `write` / `read`; те же правила для Git HTTP и API
- Команды: личные user-owned команды в `/settings/teams` и `/api/teams`; участники и гранты на репо
- Доступ: гостевое чтение публичных репозиториев; запись после входа; гости видят только публичные activity, home и search; Git HTTP для чужих private - 404
- Git по HTTP: clone, fetch, push; защита веток только fast-forward; Basic auth для private
- Просмотр: ветки, теги, дерево, blob, raw-скачивание, коммиты, diff, blame, summary (readme и статистика)
- Issues: открытие, закрытие, редактирование, удаление, комментарии, метки; монотонные номера на репозиторий
- Pull requests: открытие, закрытие, редактирование, удаление, комментарии, diff, коммиты, merge
- Зеркалирование: pull и push по http(s) и SSH; расписание, пауза, ручной sync; создание Import vs Mirror; админ-страница «Зеркала» с массовым sync, группами и verify
- Поиск репозиториев по имени; публичная главная
- Лента активности: push, issues, pull requests, создание репозитория
- YAML-конфиг (вложенный `http` / `server` / `secrets` / `auth` / `mirror`): адрес, хранилище, имя сайта, cookies, сессии, регистрация, proxies, лимиты receive и параметры зеркал
- Git HTTP push: анонимный receive-pack отвечает `401` Basic challenge (в т.ч. для private); write - только owner или collaborator/team с write
- UI на английском и русском
- Security: rate limiting, CSRF, CSP (+ лимит CSP-report по IP), HSTS; Argon2id; step-up пароль для админ-действий, удаления и revoke-others; admin-удаление пользователя держит имена репо; `git:write` PAT требует write ACL; mirror `insecure_tls` требует pin хостов и same-host redirect; trust-пути только под `data_dir`; receive по умолчанию 256 MiB / лимит зеркала 5 GiB; настраиваемый TTL сессий
- Обслуживание: очистка просроченных сессий/токенов, старой activity и soft-deleted репозиториев; bootstrap схемы SQLite на пустой БД; CLI для запуска сервера и создания пользователей
