-- +goose Up

ALTER TABLE users
    ADD COLUMN language text NOT NULL DEFAULT 'en';

-- +goose Down

ALTER TABLE users
    DROP COLUMN language;
