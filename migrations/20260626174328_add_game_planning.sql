-- +goose Up
ALTER TABLE games ADD COLUMN published boolean NOT NULL DEFAULT true;
ALTER TABLE games ADD COLUMN theme text;
ALTER TABLE games ADD COLUMN requirements text;
ALTER TABLE games_services ADD COLUMN status text NOT NULL DEFAULT 'planning';

-- +goose Down
ALTER TABLE games_services DROP COLUMN status;
ALTER TABLE games DROP COLUMN requirements;
ALTER TABLE games DROP COLUMN theme;
ALTER TABLE games DROP COLUMN published;
