CTF01D Training Platform

Прототип на Rails для демонстрации основных сущностей и потоков.

Документация
- `docs/CONCEPT.md`: концепция платформы и ключевые идеи.
- `docs/MVP_STATUS.md`: состав и статус MVP, что реализовано.
- `docs/JURY_AND_SERVICE.md`: модель «жюри/сервис», роли и взаимодействия.
- `docs/ROADMAP.md`: план работ и ближайшие шаги.

Подробности и прочие материалы смотрите в директории `docs/`.

Производственный запуск (Docker Compose)
- Требуется: Docker 24+, Docker Compose v2.
- Подготовка:
  - Скопируйте `.env.sample` в `.env` и задайте значения:
    - `POSTGRES_PASSWORD` — пароль БД
    - `RAILS_MASTER_KEY` — ключ из `config/master.key`
    - `ACME_EMAIL` — e‑mail для регистрации сертификата Let's Encrypt
- Запуск/обновление:
  - Первый запуск: `docker compose up -d --build`
  - Обновление образа: `docker compose pull && docker compose up -d --build`
- Миграции:
  - При старте веб-приложения запускается `rails db:prepare` (создание/миграция БД).
  - Для ручного прогона миграций: `docker compose run --rm app ./bin/rails db:migrate`.

Сервисы в Compose
- `db` — PostgreSQL (создаются БД: `web_rails_production`, `web_rails_production_cache`, `web_rails_production_queue`).
- `app` — Rails-приложение из `Dockerfile` (экспонирует порт `80` внутри сети Compose).
- `reverse-proxy` — Caddy с автоматическим HTTPS (Let's Encrypt), слушает `80/443` и проксирует на `app:80`.

HTTPS (Let's Encrypt)
- Настройте DNS: `A` (и при необходимости `AAAA`) запись домена `ctf01d-training-platform.ru` на IP вашего VDS.
- Укажите e‑mail для ACME в `.env` через `ACME_EMAIL`.
- При запуске `reverse-proxy` выпустит и продлит сертификаты автоматически. Логи можно смотреть: `docker compose logs -f reverse-proxy`.
