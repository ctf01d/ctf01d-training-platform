# План миграции: Ruby on Rails → Go (OpenAPI-first)

Перенос платформы **ctf01d-training-platform** с Rails 8.1 (серверный рендеринг, ERB+jQuery)
на стек **Go + Gin + PostgreSQL**, построенный вокруг единого контракта **OpenAPI**, по образцу
проекта `sendsay-open-api` (Makefile — точка входа всего пайплайна).

## 1. Зафиксированные решения

| Тема | Решение |
|------|---------|
| Backend | Go 1.26, Gin |
| Источник истины API | OpenAPI 3.0, spec-first; `oapi-codegen` (models + gin) |
| Архитектура | Лёгкая слоёная: `handler → service → repository`, группировка по сущностям |
| БД | PostgreSQL, `pgx` + `sqlc` (типобезопасный SQL-кодоген) |
| Миграции | `goose` (Go-native, sql-миграции) |
| Авторизация | JWT (Bearer), пароли — bcrypt (переносятся из `password_digest`) |
| Frontend | SPA + TypeScript-клиент, сгенерированный из `openapi.yaml` |
| Конфиг / логгер | `cleanenv` + `zap` (как в эталоне) |
| Деплой | Docker Compose + Caddy reverse-proxy (как сейчас) |

### Открытые рекомендации (можно скорректировать)
- **SPA-фреймворк:** рекомендую **React + Vite + TypeScript**; TS-клиент — `openapi-typescript`
  (типы) + `openapi-fetch` (рантайм). Альтернатива — `orval`/RTK Query, если нужен React Query «из коробки».
- **ID:** сохраняем `bigint` (как в текущей схеме), чтобы перенести данные 1:1.
  CONCEPT упоминает UUID для пользователей — при необходимости добавим публичный UUID-столбец
  отдельной миграцией, не ломая FK.
- **Перенос данных:** одноразовый скрипт миграции из текущей prod-БД (схема почти не меняется).

## 2. Целевая структура репозитория

```
.
├── Makefile                     # точка входа: build / openapi-* / sqlc / migrate / lint / test
├── api/
│   ├── openapi.yaml             # собранная спека (генерируется из фрагментов)
│   └── fragments/               # рукописные фрагменты OpenAPI
│       ├── api.schema.yaml      # info, servers, security
│       ├── components.schema.yaml  # общие схемы (Error, Pagination, …)
│       ├── users.yaml           # paths + schemas по сущности
│       ├── teams.yaml
│       ├── games.yaml
│       ├── services.yaml
│       └── …
├── gen/
│   └── httpserver/httpserver.gen.go   # oapi-codegen: модели + ServerInterface (gin)
├── cmd/
│   └── main.go                  # bootstrap: config, logger, pool, router, graceful shutdown
├── internal/
│   ├── config/                  # cleanenv-конфиг
│   ├── server/                  # сборка gin-движка, middleware, регистрация роутов
│   │   ├── server.go
│   │   ├── middleware/          # auth(JWT), requestid, cors, recovery, logging, rbac
│   │   └── handler/             # реализация ServerInterface: разбор/валидация → service
│   │       ├── users.go
│   │       ├── teams.go
│   │       └── …
│   ├── service/                 # бизнес-логика по сущностям
│   │   ├── users/
│   │   ├── teams/
│   │   ├── games/
│   │   ├── services/            # импорт github/zip, скачивание архивов, чекеры
│   │   └── ctf01d/              # экспорт в формат ctf01d (порт export_zip.rb)
│   ├── repository/
│   │   ├── db/                  # sqlc-сгенерированный код (queries + models)
│   │   ├── queries/             # *.sql — исходники для sqlc
│   │   └── store.go             # обёртка пула pgx + транзакции
│   ├── auth/                    # выдача/проверка JWT, bcrypt
│   └── domain/                  # доменные типы/ошибки/роли (не зависят от gin/pg)
├── migrations/                  # goose *.sql (создаются из текущего schema.rb)
├── pkg/                         # переиспользуемое (logger, errors, …)
├── web/                         # SPA (React+Vite+TS)
│   ├── src/
│   └── src/api/                 # сгенерированный TS-клиент из openapi.yaml
├── configs/                     # golangci, sqlc.yaml, spectral
├── docker/ , docker-compose*.yml , deploy/
└── sqlc.yaml
```

## 3. Пайплайн OpenAPI (Makefile-таргеты)

По образцу эталона, но без textile-источника — фрагменты пишем руками.

- `openapi-merge` — слить `api/fragments/*.yaml` (через `yq`) в `api/openapi.yaml`, отформатировать (`openapi-format`).
- `openapi-lint` — `spectral lint api/openapi.yaml`.
- `openapi-codegen` — `oapi-codegen -generate models,gin -o gen/httpserver/httpserver.gen.go` →
  типы запросов/ответов + `ServerInterface`, который реализуют хендлеры.
- `sqlc-gen` — `sqlc generate` из `internal/repository/queries/*.sql`.
- `web-client-gen` — `openapi-typescript api/openapi.yaml -o web/src/api/schema.d.ts`.
- `app-build` — сборка бинаря с version/buildtime через ldflags.
- `migrate-up` / `migrate-down` / `migrate-new` — goose.
- `lint` / `test` — golangci-lint + `go test ./...`.

Единый граф: **fragments → merge → (oapi-codegen для Go) + (openapi-typescript для SPA)**.
Контракт один, обе стороны генерируются из него — это и есть «фронт и бэк по openapi».

## 4. Перенос модели данных

Схема переносится из `db/schema.rb` практически 1:1 в goose-миграции. Таблицы:
`users, teams, team_memberships, team_membership_events, universities, games, game_teams,
services, results, final_results, writeups, games_services (join)`.
Сохраняем индексы, уникальные ограничения (`users.user_name`, `teams.captain_id` partial unique,
`game_teams (game_id, team_id)` и т.д.), внешние ключи, `jsonb`-поля
(`game_teams.ctf01d_overrides`, `services.ctf01d_training`).

`created_at/updated_at` — `updated_at` обновляем в репозитории/триггером (нет ActiveRecord-магии).

## 5. Карта API (Rails routes → REST-эндпоинты)

Текущие server-rendered действия превращаем в JSON-эндпоинты:

| Ресурс | Эндпоинты |
|--------|-----------|
| Auth | `POST /session` (login→JWT), `DELETE /session` (logout) |
| Profile | `GET/PATCH /profile` |
| Users | `GET/POST /users`, `GET/PATCH/DELETE /users/{id}` |
| Universities | CRUD `/universities` |
| Teams | CRUD `/teams` + `POST /teams/{id}/join-request`, `POST /teams/{id}/invite` |
| Team memberships | CRUD `/team-memberships` + `approve/reject/accept/decline/set-role` |
| Games | CRUD `/games` + `services` (list/add/remove), `finalize/unfinalize`, `export-ctf01d(-options)` |
| Game teams | `POST/PATCH/DELETE /game-teams` |
| Services | CRUD `/services` + `toggle-public`, `check-checker`, `redownload`, `upload-archives`, `download-local`, `import/github`, `import/zip` |
| Results | CRUD `/results` |
| Writeups | `POST/DELETE /writeups` |
| Scoreboard | `GET /scoreboard` |

Каждый эндпоинт описывается во фрагменте OpenAPI (request/response schema, security, ошибки).

## 6. Что требует особого внимания (нетривиальная бизнес-логика)

Это «толстые» части текущего проекта — закладываем под них отдельные итерации:
- `app/services/ctf01d/export_zip.rb` (~635 строк) → `internal/service/ctf01d` — генерация zip для жюри ctf01d.
- `app/services/service_archives.rb`, `archive_downloader.rb` → скачивание/хранение архивов сервисов
  (поля `*_local_path/size/sha256/downloaded_at`), отдача файлов, проверка sha256.
- `github_importer.rb`, `service_import/*` (bundle_builder, metadata_extractor) → импорт сервисов из GitHub/zip.
- `checker_inspector.rb` → запуск/инспекция чекеров (`check_status`, `checked_at`).
- Загрузка файлов (multipart) и файловое хранилище (`storage/`) → определить: локальный том vs S3.
- RBAC: роли `guest/player/admin` → middleware + проверки во `service`-слое.

## 7. Фронтенд (SPA)

- React + Vite + TypeScript в `web/`.
- API-клиент генерируется из `api/openapi.yaml` (`make web-client-gen`) — типобезопасные вызовы.
- Auth: JWT в памяти + refresh (или httpOnly-cookie для refresh — уточним при реализации).
- Перенос текущих экранов (games, services, teams, scoreboard, profile, …) — соответствуют папкам `app/views/*`.
- Сборка SPA → статика, раздаётся Caddy (или встраивается в бинарь через `embed` — на выбор).

## 8. Поэтапный план работ

1. **Каркас.** go.mod, структура, Makefile, конфиг (cleanenv), логгер (zap), pgx-пул, graceful shutdown,
   `/healthz`. Docker Compose с Postgres.
2. **OpenAPI-пайплайн.** Базовые фрагменты (api/components), `oapi-codegen` + `openapi-typescript`, заглушка-роутер.
3. **БД.** goose-миграции из `schema.rb`, sqlc-конфиг, базовые queries, скрипт переноса данных из prod.
4. **Auth + Users.** JWT, bcrypt, login/logout, профиль, CRUD пользователей, RBAC-middleware. End-to-end вертикальный срез как образец.
5. **Справочники и команды.** universities, teams, team_memberships (+events), invite/join/approve-логика.
6. **Игры.** games, game_teams, results, final_results, finalize/unfinalize, scoreboard.
7. **Сервисы (ядро).** CRUD, public-флаг, импорт github/zip, скачивание архивов, чекеры, отдача файлов.
8. **Экспорт ctf01d.** Порт `export_zip` + опции экспорта.
9. **SPA.** React-приложение поверх готового API; перенос экранов.
10. **Прод-обвязка.** Dockerfile (multi-stage), compose-prod, CI (golangci-lint, sqlc-diff, тесты, openapi-lint), деплой.

## 9. Тестирование
- Unit: service-слой (бизнес-правила, RBAC).
- Integration: repository против Postgres (testcontainers или dockerized PG в CI).
- Contract: `spectral` на спеку; проверка, что хендлеры реализуют сгенерированный `ServerInterface` (компиляция).
- E2E (smoke): ключевые сценарии через HTTP.

## 10. Открытые вопросы к согласованию перед стартом
1. SPA-фреймворк: React+Vite (рекомендация) или другой?
2. Файловое хранилище архивов: локальный том или S3-совместимое?
3. Перенос боевых данных нужен сразу или стартуем с чистой БД + сидов?
4. ID пользователей: оставляем bigint или вводим UUID (как в CONCEPT)?
5. Чекеры/экспорт ctf01d — переносим в первой версии или выносим во вторую фазу (MVP без них)?
