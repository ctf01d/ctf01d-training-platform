-- +goose Up
ALTER TABLE services ADD COLUMN ports integer[] NOT NULL DEFAULT '{}';
ALTER TABLE services ADD COLUMN tech_stack text[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE services DROP COLUMN tech_stack;
ALTER TABLE services DROP COLUMN ports;
