-- +goose Up
ALTER TABLE services ADD COLUMN source_kind text NOT NULL DEFAULT 'manual';
ALTER TABLE services ADD COLUMN git_repo_url text;
ALTER TABLE services ADD COLUMN git_ref text;
ALTER TABLE services ADD COLUMN git_subdir text;
ALTER TABLE services ADD COLUMN git_last_commit text;
ALTER TABLE services ADD COLUMN git_synced_at timestamptz;
ALTER TABLE services ADD COLUMN git_sync_status text NOT NULL DEFAULT 'unknown';
ALTER TABLE services ADD COLUMN git_sync_error text;

-- +goose Down
ALTER TABLE services DROP COLUMN git_sync_error;
ALTER TABLE services DROP COLUMN git_sync_status;
ALTER TABLE services DROP COLUMN git_synced_at;
ALTER TABLE services DROP COLUMN git_last_commit;
ALTER TABLE services DROP COLUMN git_subdir;
ALTER TABLE services DROP COLUMN git_ref;
ALTER TABLE services DROP COLUMN git_repo_url;
ALTER TABLE services DROP COLUMN source_kind;
