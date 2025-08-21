#!/usr/bin/env bash
set -euo pipefail

# Create additional databases for cache and queue on first run.
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-SQL
  DO $$
  BEGIN
    PERFORM 1 FROM pg_database WHERE datname = 'web_rails';
    IF NOT FOUND THEN
      CREATE DATABASE web_rails OWNER "$POSTGRES_USER";
    END IF;

    PERFORM 1 FROM pg_database WHERE datname = 'web_rails_production_queue';
    IF NOT FOUND THEN
      CREATE DATABASE web_rails_production_queue OWNER "$POSTGRES_USER";
    END IF;
  END
  $$;
SQL

