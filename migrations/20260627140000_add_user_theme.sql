-- +goose Up

ALTER TABLE users
    ADD COLUMN theme text NOT NULL DEFAULT 'classic';

-- +goose Down

ALTER TABLE users
    DROP COLUMN theme;
