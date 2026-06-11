# Database Migrations

This directory contains SQL migrations managed by [goose](https://github.com/pressly/goose).

## Naming Convention

Goose generates migration files with the format:

```
YYYYMMDDHHMMSS_<name>.up.sql
YYYYMMDDHHMMSS_<name>.down.sql
```

- Use `snake_case` for the migration name.
- Choose descriptive names: `create_users`, `add_index_on_teams_name`, etc.
- Each `up` file must have a corresponding `down` file that fully reverses it.

## Creating a New Migration

```bash
make migrate-new name=create_users
```

Or directly:

```bash
go run github.com/pressly/goose/v3/cmd/goose -dir migrations create create_users sql
```

## Running Migrations

```bash
# Apply all pending migrations
make migrate-up

# Rollback the last migration
make migrate-down

# Check current status
make migrate-status
```

## Rules

- Never edit an already-applied migration. Create a new one instead.
- Always provide a reversible `down` migration.
- Use `-- +goose Up` and `-- +goose Down` annotations.
- Use `timestamptz` for all timestamp columns.
- Use `bigserial` for primary keys (matching Rails schema).
