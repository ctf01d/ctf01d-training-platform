# Plan: Rewrite ctf01d-training-platform from Rails to Go (OpenAPI-first)

## Overview
Переписать платформу ctf01d-training-platform с Ruby on Rails 8.1 (server-rendered ERB+jQuery) на
Go + Gin + PostgreSQL, построенную вокруг единого контракта OpenAPI, плюс SPA на React+TypeScript,
типы которой генерируются из того же контракта. Слои: handler → service → repository. БД — pgx + sqlc,
миграции — goose, авторизация — JWT (bcrypt-пароли, совместимые с Rails password_digest).
Образец OpenAPI-пайплайна (Makefile как точка входа) — `/home/fox/project/sendsay-open-api`.

Старый Rails-код (`app/`, `config/`, `db/`, `Gemfile`) НЕ трогаем до достижения паритета; Go-код живёт
в новых директориях (`cmd/`, `internal/`, `api/`, `gen/`, `migrations/`, `web/`). Удаление Rails — в конце.

Сущности (из `db/schema.rb`): users, teams, team_memberships, team_membership_events, universities,
games, game_teams, services, results, final_results, writeups, games_services (join). ID — bigserial
(как в Rails). Таски выполняются строго по порядку — каждый опирается на предыдущие.

## Conventions
- Go module: `github.com/ctf01d/ctf01d-training-platform`, Go 1.26. API под префиксом `/api/v1`.
- Слои: `internal/server/handler` (HTTP, реализует сгенерированный `ServerInterface`) →
  `internal/service/<entity>` (бизнес-логика, не знает про gin) → `internal/repository` (sqlc+pgx).
- JSON-поля — snake_case (как в БД: user_name, display_name, created_at). Пути — kebab-case
  (`/team-memberships`). Время — timestamptz / RFC3339.
- Глобальные роли пользователя: `guest` (default), `player`, `admin`. Роли в команде:
  `owner, captain, vice_captain, player, guest`; управляющие: `owner, captain, vice_captain`.
  Статусы членства: `pending, approved, rejected`.
- Доменные ошибки — в `internal/domain/errs` (`ErrNotFound, ErrConflict, ErrForbidden, ErrUnauthorized,
  ErrValidation`); хендлеры мапят их в HTTP-коды единым хелпером.
- НЕ редактировать сгенерированные файлы вручную: `gen/httpserver/*.gen.go`, `internal/repository/db/*`
  (sqlc), `web/src/api/schema.d.ts`. Менять только источники (фрагменты OpenAPI / *.sql) и регенерировать.
- Сгенерированные файлы коммитим в репозиторий. Коммиты — на английском, conventional commits.
- Каждый Go-таск завершается зелёными `go build ./...`, `go vet ./...`, `go test ./...`.

## Validation Commands
- `go build ./...`
- `go vet ./...`
- `go test ./...`

### Task 1: Initialize Go module and base layout
- [x] `go mod init github.com/ctf01d/ctf01d-training-platform` (Go 1.26).
- [x] Создать каталоги с `.gitkeep`: `cmd/server/`, `internal/config/`, `internal/server/`,
      `internal/server/handler/`, `internal/server/middleware/`, `internal/service/`, `internal/repository/`,
      `internal/repository/queries/`, `internal/domain/errs/`, `pkg/logger/`, `migrations/`,
      `api/fragments/`, `gen/httpserver/`, `tools/`.
- [x] `.gitignore` для Go: `/ctf01d-server`, `*.out`, `.env`, `/tmp/`, `web/node_modules/`, `web/dist/`,
      `/storage/`. Не игнорировать существующие Rails-файлы.
- [x] `.env.sample` с переменными: `APP_ENV, HTTP_ADDR, DATABASE_URL, JWT_SECRET, JWT_TTL_HOURS,
      LOG_LEVEL, CORS_ALLOWED_ORIGINS, STORAGE_DIR, STORAGE_MAX_UPLOAD_BYTES`.

### Task 2: Configuration package
- [x] `internal/config/config.go` со структурой `Config` через `github.com/ilyakaznacheev/cleanenv`:
      `Env` (APP_ENV, default development), `HTTP.Addr` (HTTP_ADDR, default `:8080`),
      `DB.URL` (DATABASE_URL, default `postgres://postgres:postgres@localhost:5432/ctf01d_development?sslmode=disable`),
      `JWT.Secret` (JWT_SECRET), `JWT.TTLHours` (JWT_TTL_HOURS default 24),
      `Log.Level` (LOG_LEVEL default info), `CORS.AllowedOrigins` (CORS_ALLOWED_ORIGINS default
      `http://localhost:5173`), `Storage.Dir` (STORAGE_DIR default `./storage`),
      `Storage.MaxUploadBytes` (STORAGE_MAX_UPLOAD_BYTES default 209715200).
- [x] `Load() (*Config, error)`: в env `production` требует непустые JWT_SECRET и DATABASE_URL.
- [x] Тест `config_test.go` (через `t.Setenv`) на дефолты и required.

### Task 3: Logger and DB pool
- [x] `go get go.uber.org/zap`. `pkg/logger/logger.go`: `New(env, level string) (*zap.Logger, error)`
      (production→json, иначе console; уровень из level) и `Sync(l)`.
- [x] `go get github.com/jackc/pgx/v5 github.com/jackc/pgx/v5/pgxpool`.
- [x] `internal/repository/store.go`: `Store{ Pool *pgxpool.Pool }`,
      `NewStore(ctx, dbURL) (*Store, error)` (создаёт пул + Ping), `Close()`, `Health(ctx) error` (Ping).

### Task 4: HTTP server with healthz and graceful shutdown
- [x] `go get github.com/gin-gonic/gin github.com/gin-contrib/cors github.com/gin-contrib/requestid`.
- [x] `internal/server/server.go`: `New(cfg, log, store) *gin.Engine` — `gin.New()`, middleware Recovery,
      requestid, CORS (origins из cfg), zap-логгер запросов. Роуты `GET /healthz` (200 ok / 503 если
      Health падает) и `GET /version` (статическая версия).
- [x] `cmd/server/main.go`: bootstrap config→logger→store→server→`http.Server{Addr, Handler,
      ReadHeaderTimeout:5s}`; graceful shutdown по SIGINT/SIGTERM (Shutdown с таймаутом 5s, затем Close).
- [x] Тест `/healthz` через `httptest` (Health за интерфейсом `Pinger`, подменяется в тесте).

### Task 5: Makefile and dev Docker Compose
- [x] Добавить Go-таргеты в Makefile (не ломая существующий Rails Makefile; уникальные имена,
      каждый с `## name: описание`): `go-build` (`go build -o ctf01d-server ./cmd/server`), `go-run`,
      `go-test`, `go-vet`, `go-fmt` (gofmt -w + gofumpt при наличии), `go-tidy` (`go mod tidy`).
- [x] `docker-compose.dev.yml`: `db` (postgres:16, POSTGRES_USER/PASSWORD=postgres,
      POSTGRES_DB=ctf01d_development, порт 5432, volume) и опционально `adminer` (8081).
- [x] `docs/GO_DEV.md` с инструкцией локального запуска и требованиями к инструментам
      (yq v4, Node.js, golangci-lint). Выполнить `go mod tidy`, убедиться что build/test зелёные.

### Task 6: OpenAPI tooling and Makefile pipeline
- [x] `tools/tools.go` (build-тег `//go:build tools`) с blank-импортами:
      `github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen`,
      `github.com/sqlc-dev/sqlc/cmd/sqlc`, `github.com/pressly/goose/v3/cmd/goose`. `go get` для них.
- [x] Makefile-таргеты: `openapi-merge` (слить `api/fragments/**/*.yaml` через `yq eval-all
      '. as $i ireduce ({}; . * $i)'` в `api/openapi.yaml`), `openapi-codegen`
      (`go run .../oapi-codegen -config configs/oapi-codegen.yaml api/openapi.yaml`),
      `openapi-ts` (`npx openapi-typescript api/openapi.yaml -o web/src/api/schema.d.ts`),
      `openapi` (merge→codegen→ts), `openapi-lint` (`npx @stoplight/spectral-cli lint api/openapi.yaml
      --ruleset configs/spectral.yaml`).
- [x] `configs/oapi-codegen.yaml`: package httpserver, output `gen/httpserver/httpserver.gen.go`,
      generate {models:true, gin-server:true, embedded-spec:true}.
- [x] `configs/spectral.yaml`: extends `spectral:oas` (отключить шумные правила при необходимости).

### Task 7: Base OpenAPI fragments
- [x] `api/fragments/00-base.schema.yaml`: `openapi: 3.0.3`, info (title `CTF01D Training Platform API`,
      version 1.0.0), servers (`http://localhost:8080`), tags {}, paths {}, components {},
      `security: [{ BearerAuth: [] }]`.
- [x] `api/fragments/10-components.schema.yaml` → `components`: securitySchemes.BearerAuth (http/bearer/JWT);
      schemas: `Error` (code, message, details nullable), `Pagination` (page, per_page, total),
      `Timestamped` (created_at, updated_at date-time); parameters PageParam/PerPageParam (query int,
      default 1/20); responses NotFound/Unauthorized/Forbidden/ValidationError/Conflict (ref Error).

### Task 8: Users OpenAPI fragment and codegen wiring
- [x] `api/fragments/users.yaml` (tag users): схемы `User` (allOf Timestamped + id int64, user_name,
      display_name, role enum[guest,player,admin], rating int, avatar_url nullable),
      `UserCreate` (user_name required pattern `^[a-zA-Z0-9_]+$`, display_name required, password required
      minLength 6, role enum default guest, avatar_url nullable), `UserUpdate` (display_name, avatar_url,
      password — все optional), `UserList` (items[] + pagination). Пути CRUD `/users`
      (operationId listUsers/createUser/getUser/updateUser/deleteUser) с кодами 200/201/204/404/409/422.
- [x] `make openapi` (merge→codegen→ts), убедиться что создан `api/openapi.yaml`,
      `gen/httpserver/httpserver.gen.go` (с `ServerInterface` и типами) и `web/src/api/schema.d.ts`.
- [x] `internal/server/handler/handler.go`: `Handler` (пока пустой) + конструктор; реализовать ВСЕ методы
      `httpserver.ServerInterface` заглушками `501 {"code":"not_implemented"}`. Проверить компиляцией
      `var _ httpserver.ServerInterface = (*Handler)(nil)`.
- [x] В `server.go` подключить роуты через `httpserver.RegisterHandlersWithOptions(engine, handler,
      {BaseURL:"/api/v1"})`.

### Task 9: Error mapping and request helpers
- [ ] `internal/domain/errs/errs.go`: sentinel-ошибки `ErrNotFound, ErrConflict, ErrForbidden,
      ErrUnauthorized`; тип `ValidationError{ Fields map[string]string }` (реализует error).
- [ ] `internal/server/handler/response.go`: `respondError(c, err)` (errors.Is/As → 404/409/403/401/422,
      прочее → 500, тело Error), `bindJSON[T](c) (T, bool)` (422 при ошибке). Unit-тест на каждый класс.

### Task 10: goose migrations setup
- [ ] `go get github.com/pressly/goose/v3`. Makefile-таргеты: `migrate-up`
      (`goose -dir migrations postgres "$$DATABASE_URL" up`), `migrate-down`, `migrate-status`,
      `migrate-new name=...` (`goose -dir migrations create $(name) sql`).
- [ ] `migrations/README.md` с правилами именования.

### Task 11: Schema migration from schema.rb
Воспроизвести `db/schema.rb` в goose-миграции (`-- +goose Up`/`Down`; ID bigserial; timestamps
`timestamptz NOT NULL DEFAULT now()`). Down должен полностью откатывать.
- [ ] `users`: user_name (NOT NULL, unique index), display_name NOT NULL, role NOT NULL default 'guest',
      rating int NOT NULL default 0, avatar_url, password_digest, timestamps.
- [ ] `universities`: name, site_url, avatar_url, timestamps.
- [ ] `teams`: name NOT NULL, description text, website, avatar_url, captain_id bigint, university_id bigint,
      timestamps. Partial unique `teams(captain_id) WHERE captain_id IS NOT NULL`, index university_id,
      FK university_id→universities.
- [ ] `team_memberships`: team_id NOT NULL, user_id NOT NULL, role, status, timestamps; indexes team_id,
      user_id; FK teams, users.
- [ ] `team_membership_events`: team_id NOT NULL, user_id NOT NULL, actor_id int, action NOT NULL,
      from_role/to_role/from_status/to_status, timestamps; indexes actor_id, (team_id, created_at),
      team_id, user_id; FK teams, users.
- [ ] `games`: name, organizer, starts_at/ends_at, avatar_url, site_url, ctftime_url,
      finalized bool NOT NULL default false, finalized_at, registration_opens_at/closes_at,
      scoreboard_opens_at/closes_at, vpn_url, vpn_config_url, access_instructions text, access_secret,
      timestamps.
- [ ] `services`: ВСЕ поля из schema.rb — name NOT NULL (unique index), public_description/
      private_description text, author, copyright, avatar_url, public bool NOT NULL default true,
      service_archive_url, checker_archive_url, writeup_url, exploits_url,
      check_status NOT NULL default 'unknown', checked_at, service_local_path,
      service_local_size int, service_local_sha256, service_downloaded_at, checker_local_path,
      checker_local_size int, checker_local_sha256, checker_downloaded_at,
      ctf01d_training jsonb NOT NULL default '{}', timestamps.
- [ ] `games_services` (join, без id): game_id NOT NULL, service_id NOT NULL; unique (game_id, service_id),
      index (service_id, game_id); FK games, services.
- [ ] `game_teams`: game_id NOT NULL, team_id NOT NULL, ip_address, ctf01d_id,
      ctf01d_overrides jsonb NOT NULL default '{}', team_type, "order" int NOT NULL default 0
      (имя экранировать кавычками), timestamps; indexes (game_id,"order",id), unique (game_id,team_id),
      game_id, team_id; FK games, teams.
- [ ] `results`: game_id NOT NULL, team_id NOT NULL, score int, timestamps; unique (game_id,team_id),
      indexes game_id, team_id; FK games, teams.
- [ ] `final_results`: game_id NOT NULL, team_id NOT NULL, score int NOT NULL default 0, position int,
      timestamps; unique (game_id,team_id), indexes game_id, team_id; FK games, teams.
- [ ] `writeups`: game_id NOT NULL, team_id NOT NULL, title NOT NULL, url NOT NULL, timestamps;
      unique (game_id,team_id,title), indexes game_id, team_id; FK games, teams.
- [ ] Триггерная функция `set_updated_at()` + триггеры на все таблицы с updated_at.
- [ ] Прогнать `make migrate-up` и `make migrate-down` против dev-БД — оба чистые.

### Task 12: sqlc configuration and store transactions
- [ ] `sqlc.yaml`: version 2, engine postgresql, schema `migrations`, queries `internal/repository/queries`,
      gen.go: package db, out `internal/repository/db`, sql_package pgx/v5, emit_json_tags true,
      emit_pointers_for_null_types true; overrides timestamptz→time.Time, jsonb→json.RawMessage.
      Makefile-таргеты `sqlc-gen` и `sqlc-vet`.
- [ ] Расширить `store.go`: встроить `*db.Queries`, метод `WithTx(ctx, fn func(*db.Queries) error) error`
      (Begin→WithTx→Commit/Rollback).
- [ ] Helper `internal/repository/testhelper_test.go`: подключение к `TEST_DATABASE_URL` (иначе t.Skip),
      применение goose-миграций programmatically, очистка `TRUNCATE ... RESTART IDENTITY CASCADE`.

### Task 13: Users queries, service and JWT/bcrypt
- [ ] `internal/repository/queries/users.sql` (аннотации sqlc): CreateUser, GetUserByID,
      GetUserByUserName, ListUsers(limit,offset), CountUsers, UpdateUserProfile (display_name,
      avatar_url, password_digest nullable через COALESCE), UpdateUserRole, UpdateUserRating, DeleteUser.
      `make sqlc-gen`.
- [ ] `go get golang.org/x/crypto/bcrypt github.com/golang-jwt/jwt/v5`.
- [ ] `internal/auth/password.go`: `HashPassword(plain)` (bcrypt cost 12), `CheckPassword(hash, plain) bool`.
- [ ] `internal/auth/jwt.go`: `Claims` (jwt.RegisteredClaims + Role, UserName), `Manager{secret, ttl}`,
      `NewManager(secret, ttl)`, `Generate(userID, role, userName) (string, error)`,
      `Parse(token) (*Claims, error)` (HS256, валидация подписи и exp). Тесты round-trip + протухший/битый.
- [ ] `internal/service/users/service.go`: `Service{q, store}`; методы Create (валидация user_name regex,
      хеш пароля, uniqueness→ErrConflict), GetByID, List(page,perPage)→(items,total), Update, UpdateRole,
      Delete. Доменный тип `User` (password_digest наружу не отдавать). Unit-тесты с мок-Querier.

### Task 14: Auth fragment, service, handlers and RBAC middleware
- [ ] `api/fragments/auth.yaml` (tag auth): `LoginRequest` (user_name, password), `LoginResponse`
      (token, user $ref User). Пути `POST /session` (login, public, 200/401),
      `DELETE /session` (logout, 204), `GET /profile` (200/401), `PATCH /profile` (UserUpdate без role,
      200/401/422). `make openapi`.
- [ ] `internal/service/auth/service.go`: `Login(ctx, userName, password) (token, User, err)`
      (найти→bcrypt→ErrUnauthorized или JWT), `Me(ctx, userID) (User, error)`; logout — stateless.
- [ ] `internal/server/middleware/auth.go`: `RequireAuth(jwtMgr)` (читает Bearer, парсит, 401 при ошибке,
      кладёт user_id/role/user_name в контекст; хелперы CurrentUserID/CurrentRole),
      `RequireRole(role)` (403 при недостатке; иерархия guest<player<admin), `OptionalAuth`.
      Тесты: нет токена→401, битый→401, валидный→контекст, мало прав→403.
- [ ] `internal/server/handler/auth.go` + `users.go`: реализовать login/logout/getProfile/updateProfile
      и users CRUD; убрать соответствующие заглушки. Роуты: `/session` POST public, всё под `/api/v1` —
      RequireAuth; create/update(role)/delete users — RequireRole("admin"); свой профиль — без admin.
- [ ] Прокинуть Users/Auth/JWT в Handler и main. Интеграционный тест (skip без TEST_DATABASE_URL):
      seed admin→login→создать пользователя→list→get→update→смена роли→delete; 401 без токена.

### Task 15: Universities and Teams fragments + queries
- [ ] `api/fragments/universities.yaml`: University, UniversityCreate/Update/List; CRUD `/universities`.
- [ ] `api/fragments/teams.yaml`: Team (id, name, description, website, avatar_url, captain_id nullable,
      university_id nullable, timestamps), TeamCreate/Update/List; CRUD `/teams` + `POST /teams/{id}/join-request`,
      `POST /teams/{id}/invite` (body {user_id}), `GET /teams/{id}/members`, `GET /teams/{id}/events`.
- [ ] `api/fragments/team-memberships.yaml`: TeamMembership (id, team_id, user_id, role, status, timestamps),
      TeamMembershipCreate/Update/List, SetRoleRequest (role enum), TeamMembershipEvent. Пути CRUD
      `/team-memberships` + `POST /team-memberships/{id}/approve|reject|accept|decline|set-role`. `make openapi`.
- [ ] Queries: `universities.sql` (CRUD), `teams.sql` (CRUD + GetTeamByCaptain, SetCaptain, ClearCaptain),
      `team_memberships.sql` (Create/GetByID/Update/Delete, ListByTeam, ListByUser, GetMembership(team,user),
      UpdateMembershipStatus, UpdateMembershipRole, CountApprovedManagers(team)),
      `team_membership_events.sql` (CreateEvent, ListByTeam). `make sqlc-gen`.

### Task 16: Universities, Teams, Memberships services
- [ ] `internal/service/universities/service.go`: CRUD (мутации — admin, проверка в handler). Тесты.
- [ ] `internal/service/teams/service.go`: CRUD; Create — создатель становится owner (membership
      role=owner/status=approved + событие); captain_id глобально уникален→ErrConflict;
      Update/Delete — admin или управляющий член (`CanManage(ctx, teamID, userID, role)`);
      `RequestJoin` (membership guest/pending + событие join_request); `Invite` (управляющий → membership
      player/pending + событие invite). Мутации членства в транзакции (store.WithTx). Unit-тесты:
      owner-flow, captain uniqueness, матрица прав управления.
- [ ] `internal/service/memberships/service.go`: List/Get/Create/Update/Delete; Approve/Reject
      (pending→approved/rejected, событие; управляющий/admin); Accept/Decline (для приглашённого);
      SetRole (нельзя удалить/понизить последнего owner; назначение captain синхронит teams.captain_id
      с учётом глобальной уникальности). Каждое изменение — событие с from/to role/status + actor_id,
      в транзакции. Unit-тесты на переходы и защиту последнего owner.

### Task 17: Universities/Teams/Memberships handlers + integration
- [ ] Хендлеры `universities.go`, `teams.go`, `team_memberships.go`: реализовать все методы, actor из
      контекста, мапить доменные ошибки, убрать заглушки. Прокинуть сервисы в Handler и main.
- [ ] RBAC: чтение — авторизованным; мутации вузов — admin; мутации команд/членства — внутри сервиса по
      управляющей роли (handler передаёт actor).
- [ ] Интеграционный тест: пользователи+команда→owner приглашает→accept→set-role captain (проверка
      captain_id)→сторонний join-request→approve→список членов и события→управление не-управляющим→403.

### Task 18: Games status logic, fragments and queries
- [ ] `internal/service/games/status.go`: чистые функции `ComputeStatus(startsAt,endsAt,now)`
      (upcoming/ongoing/past/unknown), `ComputeRegistrationStatus(opens,closes,now)`
      (unscheduled/upcoming/open/closed), `ComputeScoreboardStatus(opens,closes,now)`
      (always/upcoming/open/closed) — портировать из `app/models/game.rb` 1:1. Unit-тесты всех веток.
- [ ] `api/fragments/games.yaml`: Game (все поля schema.rb + read-only status/registration_status/
      scoreboard_status; access_secret отдавать только admin/управляющим), GameCreate/Update/List;
      CRUD `/games` + `GET/POST /games/{id}/services`, `DELETE /games/{id}/services/{service_id}`,
      `POST /games/{id}/finalize`, `POST /games/{id}/unfinalize`.
- [ ] `api/fragments/game-teams.yaml`: GameTeam (id, game_id, team_id, ip_address, ctf01d_id,
      ctf01d_overrides object, team_type, order, timestamps), GameTeamCreate/Update/List; пути
      `GET /games/{id}/teams`, `POST /game-teams`, `PATCH /game-teams/{id}`, `DELETE /game-teams/{id}`,
      `POST /games/{id}/teams/reorder` (body [{id,order}]).
- [ ] `api/fragments/results.yaml`: Result, ResultCreate/Update/List; CRUD `/results`.
- [ ] `api/fragments/scoreboard.yaml`: ScoreboardEntry (team_id, team_name, score, position),
      Scoreboard (game_id, status, entries[]); `GET /games/{id}/scoreboard` (public, с учётом окна) и
      `GET /scoreboard` (общий рейтинг). `make openapi`.
- [ ] Queries: `games.sql` (CRUD + SetFinalized), `games_services.sql` (AddService ON CONFLICT DO NOTHING,
      RemoveService, ListServicesByGame), `game_teams.sql` (CRUD + ListByGame ORDER BY "order",id +
      UpdateOrder), `results.sql` (CRUD + ListByGame ORDER BY score DESC + UpsertResult),
      `final_results.sql` (DeleteByGame, InsertFinalResult, ListByGame). `make sqlc-gen`.

### Task 19: Games, game-teams, results, scoreboard services
- [ ] `internal/service/games/service.go`: CRUD + URL-валидация (validHTTPURL для site_url/ctftime_url/
      vpn_url/vpn_config_url); AddService/RemoveService/ListServices;
      `Finalize(ctx, gameID)` — в транзакции: finalized=true, finalized_at=now, удалить старые
      final_results, пересчитать из results (ORDER BY score DESC → position 1..N, ties как в Rails
      games_controller); `Unfinalize` (finalized=false, finalized_at=null, удалить final_results).
      Маппинг Game→DTO добавляет вычисляемые статусы. Unit-тесты finalize/unfinalize/URL.
- [ ] `internal/service/gameteams/service.go`: CRUD ростера (uniqueness (game_id,team_id)→conflict),
      `Reorder(ctx, gameID, []{id,order})` в транзакции.
- [ ] `internal/service/results/service.go`: CRUD + upsert по (game_id,team_id); запрет правок при
      finalized (только admin) — повторить поведение Rails.
- [ ] `internal/service/scoreboard/service.go`: `ForGame(ctx, gameID, viewerRole)` (finalized→final_results,
      иначе results; для не-admin при closed/upcoming окне→ErrForbidden), `Global(ctx)`.

### Task 20: Games handlers, RBAC, secret fields + integration
- [ ] Хендлеры `games.go`, `game_teams.go`, `results.go`, `scoreboard.go`; убрать заглушки.
- [ ] RBAC: мутации игр/ростера/результатов — RequireRole("player"); чтение — авторизованным;
      `GET /games/{id}/scoreboard` — public (OptionalAuth, viewerRole). `access_secret`/VPN-поля —
      только admin или approved-членам команд-участниц (портировать `can_access_game?` из
      ApplicationController). Прокинуть сервисы в Handler и main.
- [ ] Интеграционный тест: игра→команды в ростер→reorder→сервисы→результаты→finalize (проверка
      final_results/позиций)→scoreboard→unfinalize; сокрытие access_secret для постороннего; закрытый scoreboard.

### Task 21: Storage abstraction and services queries
- [ ] `internal/storage/storage.go`: интерфейс `Storage` (Save(ctx,key,r)→FileInfo, Open(ctx,key),
      Delete(ctx,key), Stat(ctx,key)→FileInfo), `FileInfo{Size int64; SHA256 string}`.
- [ ] `internal/storage/local.go`: `LocalStorage` поверх cfg.Storage.Dir; Save стримит в файл, считая
      sha256+размер; ключи вида `services/{id}/service.zip`; защита от path traversal. Unit-тесты.
- [ ] `api/fragments/services.yaml`: Service (все поля; private_description и локальные абсолютные пути
      скрывать от не-admin; локальные архивы как метаданные {size, sha256, downloaded_at}),
      ServiceCreate/Update/List, GithubImportRequest (repo_url, опц. ref/subdir), ImportResult
      (service, warnings[]). Пути: CRUD `/services` (фильтр ?public=true, поиск ?q=);
      `POST /services/{id}/toggle-public`, `POST /services/{id}/check-checker`,
      `POST /services/{id}/redownload`, `POST /services/{id}/upload-archives` (multipart service_archive/
      checker_archive), `GET /services/{id}/download/{kind}` (service|checker, application/zip),
      `POST /services/import/github`, `POST /services/import/zip` (multipart). `make openapi`.
- [ ] `internal/repository/queries/services.sql`: Create/GetByID/GetByName/List(public,q)/Count/Update/
      Delete; SetPublic; SetCheckStatus(id,status,checked_at); SetServiceLocal(id,path,size,sha256,
      downloaded_at); SetCheckerLocal(...); SetArchiveURLs. `make sqlc-gen`.

### Task 22: Services core, archives, download
- [ ] `internal/service/services/service.go`: CRUD + URL-валидация (порт `Service#validate_urls`:
      avatar_url допускает data:image, остальные валидный http(s)://); TogglePublic; маппинг скрывает
      private_description и локальные пути от не-admin; uniqueness name→conflict. Unit-тесты.
- [ ] `internal/service/services/archives.go` (порт `archive_downloader.rb` + `service_archives.rb`):
      `Redownload(ctx,id)` (скачать service/checker архивы по URL с таймаутом, лимитом размера, проверкой
      zip; сохранить через Storage; записать local path/size/sha256/downloaded_at);
      `UploadArchives(ctx,id,serviceFile,checkerFile)` (сохранить загруженные, обновить метаданные);
      `OpenLocal(ctx,id,kind)→(rc,name,err)`. Лимит размера (cfg.Storage.MaxUploadBytes), валидация zip.
      Unit-тесты с httptest и in-memory storage.

### Task 23: Services imports and checker inspection
- [ ] `internal/service/services/import_common.go`: безопасная распаковка zip (лимит суммарного размера и
      числа файлов, запрет путей с `..`, anti zip-bomb по коэффициенту сжатия), извлечение метаданных,
      сборка bundle.
- [ ] `import_github.go` (порт `github_importer.rb` + `service_import/*`): `ImportFromGithub(ctx,req)→
      (Service, warnings[], err)` — скачать репозиторий (codeload zip), извлечь метаданные (порт
      `metadata_extractor.rb`: name, descriptions, author, ctf01d_training), собрать архивы (порт
      `bundle_builder.rb`), создать/обновить Service + сохранить архивы; накапливать warnings.
- [ ] `import_zip.go`: `ImportFromZip(ctx, zipFile)→(Service, warnings[], err)` аналогично, источник —
      загруженный zip.
- [ ] `checker.go` (порт `checker_inspector.rb`): `CheckChecker(ctx,id)` — статическая проверка структуры
      локального архива чекера, выставить check_status (ok/failed/unknown) + checked_at (динамический запуск —
      TODO-комментарий). Фикстуры в `testdata/`. Unit-тесты на извлечение метаданных, path-traversal, структуру.

### Task 24: Services handlers + integration
- [ ] `internal/server/handler/services.go`: CRUD, toggle, check, redownload, upload (c.FormFile),
      download (стрим файла с Content-Type/Content-Disposition), import github/zip. Убрать заглушки.
- [ ] RBAC: мутации/импорт/чек — RequireRole("player"); download публичного — авторизованным,
      непубличного — player/admin. Прокинуть Storage и сервисы в Handler и main.
- [ ] Интеграционный тест: create→upload-archives→download→toggle-public→list(public/q)→import из
      тестового zip→check-checker.

### Task 25: ctf01d export fragment, params and pure exporter
- [ ] Расширить `api/fragments/games.yaml`: `GET /games/{id}/export/ctf01d/options`
      (getCtf01dExportOptions → Ctf01dExportOptions: дефолты + warnings о неполных данных),
      `POST /games/{id}/export/ctf01d` (exportCtf01d, body Ctf01dExportRequest → 200 application/zip
      binary, или 422 со списком ошибок). Схема Ctf01dExportRequest (prefix, include_html, html_source_path,
      include_compose, compose_project, scoreboard.port/htmlfolder/random, overrides). `make openapi`.
- [ ] `internal/service/ctf01d/types.go` — зеркало входа Ruby-сервиса: GameParams (id, name, start_utc,
      end_utc, coffee_break_start/end_utc, flag_ttl_min, basic_attack_cost, defence_cost),
      ScoreboardParams (port, htmlfolder, random), TeamParams (id, name, active, ip_address, logo_rel,
      logo_src), CheckerParams (id, name, enabled, script_wait, round_sleep, script_rel, files[]{src,rel}),
      Options (prefix, include_html, html_source_path, include_compose, compose_project); `ExportError`
      (накопление сообщений).
- [ ] `internal/service/ctf01d/exporter.go` — порт `app/services/ctf01d/export_zip.rb` пошагово, сохраняя
      поведение: `Export(params)→(filename, data []byte, err)`; hydrateCheckersFromBundles +
      detectCheckerEntrypoint (checker.*/checker на верхнем уровне, иначе basename); validateInputs
      (собрать все ошибки: обязательные поля, уникальность id, корректность ip); buildYAMLConfig
      (config.yml с заголовком-комментарием, gopkg.in/yaml.v3, детерминированный порядок ключей);
      ensureTeamLogos (из файла / data:URL / http(s) с таймаутом / fallback SVG; согласование расширения
      с logo_rel); materializeCheckers (checker_<id>/, dummy при отсутствии); materializeServiceArchives
      (bundle service/+checker/); buildComposeYML (опционально); сборка zip из временного каталога через
      archive/zip в память. Лимиты на скачивание/распаковку, проверка путей.

### Task 26: ctf01d builder, handlers, tests
- [ ] `internal/service/ctf01d/builder.go`: `BuildParams(ctx, gameID, req)→(GameParams, ScoreboardParams,
      []TeamParams, []CheckerParams, Options, warnings[], err)` — собрать из БД (game времена→UTC,
      game_teams→TeamParams через ip/ctf01d_id/order/overrides, связанные services→CheckerParams через
      локальные архивы + ctf01d_training/overrides);
      `BuildOptions(ctx, gameID)→(Ctf01dExportOptions, warnings, err)` (дефолты + warnings: команда без ip,
      сервис без локального архива, нет логотипа — портировать из `export_ctf01d_options`).
- [ ] `internal/server/handler/ctf01d.go`: GetCtf01dExportOptions (JSON), ExportCtf01d (builder→exporter→
      стрим zip с Content-Type application/zip + Content-Disposition; 422 с details при ошибках валидации).
      RBAC: RequireRole("player")/admin. Прокинуть сервис в Handler и main.
- [ ] Фикстуры в `internal/service/ctf01d/testdata/` (bundle-zip сервиса, png-логотип, html-каталог).
      Unit-тесты exporter: распаковать сгенерированный zip и проверить data/config.yml (валидный YAML),
      data/html, логотипы, checker_<id>/, docker-compose.yml при include_compose. Тест builder на маппинг
      overrides/training и warnings.

### Task 27: Seeds and optional data import
- [ ] `cmd/seed/main.go`: создать admin (user_name=admin, пароль из SEED_ADMIN_PASSWORD или admin12345,
      role=admin, bcrypt cost 12), тестовые вузы/команды/игру; идемпотентно (ON CONFLICT DO NOTHING).
      Makefile-таргет `seed` (`go run ./cmd/seed`).
- [ ] `cmd/import-rails/main.go` — заготовка переноса из старой Rails-БД (RAILS_DATABASE_URL→DATABASE_URL):
      реализовать users (password_digest как есть — bcrypt совместим), остальные таблицы — TODO. Тест на
      чистую функцию маппинга row→params.

### Task 28: Frontend SPA scaffold and typed client
- [ ] Создать `web/` (Vite react-ts): package.json, vite.config.ts (dev-proxy `/api`→`http://localhost:8080`),
      tsconfig.json, index.html, src/main.tsx, src/App.tsx. Зависимости: react-router-dom, openapi-fetch;
      dev: openapi-typescript, eslint, @typescript-eslint/*, prettier. Скрипты: dev, build, lint,
      typecheck (`tsc --noEmit`), gen:api (`openapi-typescript ../api/openapi.yaml -o src/api/schema.d.ts`).
- [ ] `npm run gen:api` → `src/api/schema.d.ts`. `src/api/client.ts`: `createClient<paths>({baseUrl:
      '/api/v1'})` + middleware (Authorization Bearer из стора, обработка 401→разлогин/редирект).
      Тонкие обёртки по доменам в `src/api/`.
- [ ] Makefile-таргеты: `web-install` (npm ci), `web-build`, `web-gen`, `web-dev`.

### Task 29: Frontend auth, routing and screens
- [ ] `src/auth/AuthContext.tsx`: токен (localStorage) + текущий пользователь, login (POST /session),
      logout, useAuth(). `src/routes.tsx`: react-router-dom, ProtectedRoute (редирект на /login),
      AdminRoute (по роли), Layout с навигацией. Страница Login.
- [ ] Экраны (соответствуют `app/views/*`): Games (список со статусами, детали, CRUD player/admin, ростер,
      привязка сервисов, finalize/unfinalize, кнопка экспорта ctf01d — скачивание zip); Services (каталог с
      фильтром public/поиском, детали, CRUD, toggle-public, импорт github/zip, upload-archives,
      check-checker, скачивание архивов); Teams (список, состав, события, CRUD, join-request, invite,
      approve/reject/accept/decline/set-role); Universities, Users (admin), Results, Profile, Scoreboard
      (по игре и общий). Общие компоненты: таблицы с пагинацией, формы, отображение 422 по полям,
      multipart-загрузка, обработка 401/403, состояния загрузки/пустые. Убедиться что
      `npm run build`, `npm run lint`, `npm run typecheck` зелёные.

### Task 30: Production build, CI and Rails decommission
- [ ] `docker/Dockerfile` (multi-stage): builder golang:1.26 (`go build -ldflags "-X main.version=..."`,
      CGO_ENABLED=0) → distroless/static или alpine с бинарём + `migrations/`. В `cmd/server/main.go`
      реализовать запуск goose-миграций при старте под env `RUN_MIGRATIONS=true`. EXPOSE 8080,
      healthcheck `GET /healthz`.
- [ ] Раздача SPA: стадия сборки `web/` (node:22 `npm ci && npm run build`) → Caddy отдаёт `web/dist` и
      проксирует `/api/*` на app, SPA-fallback на index.html (обновить Caddyfile).
- [ ] `docker-compose.prod.yml` (новый, не ломая старый): db (postgres:16, volume, healthcheck),
      app (Go-образ, env из .env, depends_on db healthy, restart unless-stopped), reverse-proxy
      (Caddy, 80/443, ACME). Прод-БД ctf01d_production.
- [ ] `configs/golangci.yaml` (govet, staticcheck, errcheck, gofumpt, gci, revive) + Makefile `lint`/`lint-fix`.
      Таргет `verify-codegen` (`make openapi` + `make sqlc-gen`, затем `git diff --exit-code` по gen/,
      internal/repository/db/, api/openapi.yaml, web/src/api/schema.d.ts).
- [ ] Обновить `.gitlab-ci.yml` и/или `.github/workflows/*`: стадии lint (golangci + spectral + eslint),
      codegen (verify-codegen), test (go test с postgres-сервисом и TEST_DATABASE_URL; web typecheck),
      build (docker build + web build).
- [ ] После подтверждённого паритета: удалить Rails (`app/`, Rails-части `config/`, `Gemfile*`, `Rakefile`,
      `bin/rails`, `config.ru`, старые `db/migrate`+`schema.rb`, старый `docker-compose-prod.yml`,
      Ruby-Dockerfile). Обновить корневой `README.md` под Go-стек. Финальная проверка: `make lint`,
      `go test ./...`, `docker build`, поднятие prod-compose локально.
