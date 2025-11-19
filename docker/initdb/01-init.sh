#!/usr/bin/env bash
set -euo pipefail

# Create additional databases for cache and queue on first run.
primary_db="${POSTGRES_DB:-web_rails_production}"
cache_db="${primary_db}_cache"
queue_db="${primary_db}_queue"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-SQL
  DO $$
  BEGIN
    PERFORM 1 FROM pg_database WHERE datname = '${primary_db}';
    IF NOT FOUND THEN
      CREATE DATABASE ${primary_db} OWNER "$POSTGRES_USER";
    END IF;

    PERFORM 1 FROM pg_database WHERE datname = '${cache_db}';
    IF NOT FOUND THEN
      CREATE DATABASE ${cache_db} OWNER "$POSTGRES_USER";
    END IF;

    PERFORM 1 FROM pg_database WHERE datname = '${queue_db}';
    IF NOT FOUND THEN
      CREATE DATABASE ${queue_db} OWNER "$POSTGRES_USER";
    END IF;
  END
  $$;
SQL
