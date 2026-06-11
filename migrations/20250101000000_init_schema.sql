-- +goose Up
-- goose migration: initial schema from db/schema.rb

-- Trigger function for updated_at
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- users
CREATE TABLE users (
    id bigserial PRIMARY KEY,
    user_name text NOT NULL,
    display_name text NOT NULL,
    role text NOT NULL DEFAULT 'guest',
    rating integer NOT NULL DEFAULT 0,
    avatar_url text,
    password_digest text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX index_users_on_user_name ON users (user_name);

CREATE TRIGGER set_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- universities
CREATE TABLE universities (
    id bigserial PRIMARY KEY,
    name text,
    site_url text,
    avatar_url text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER set_universities_updated_at
    BEFORE UPDATE ON universities
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- teams
CREATE TABLE teams (
    id bigserial PRIMARY KEY,
    name text NOT NULL,
    description text,
    website text,
    avatar_url text,
    captain_id integer,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    university_id bigint
);

CREATE UNIQUE INDEX index_teams_on_captain_id_unique ON teams (captain_id) WHERE captain_id IS NOT NULL;
CREATE INDEX index_teams_on_university_id ON teams (university_id);

CREATE TRIGGER set_teams_updated_at
    BEFORE UPDATE ON teams
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- team_memberships
CREATE TABLE team_memberships (
    id bigserial PRIMARY KEY,
    team_id bigint NOT NULL,
    user_id bigint NOT NULL,
    role text,
    status text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX index_team_memberships_on_team_id ON team_memberships (team_id);
CREATE INDEX index_team_memberships_on_user_id ON team_memberships (user_id);
CREATE UNIQUE INDEX index_team_memberships_on_team_id_and_user_id ON team_memberships (team_id, user_id);

CREATE TRIGGER set_team_memberships_updated_at
    BEFORE UPDATE ON team_memberships
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- team_membership_events
CREATE TABLE team_membership_events (
    id bigserial PRIMARY KEY,
    team_id bigint NOT NULL,
    user_id bigint NOT NULL,
    actor_id integer,
    action text NOT NULL,
    from_role text,
    to_role text,
    from_status text,
    to_status text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX index_team_membership_events_on_actor_id ON team_membership_events (actor_id);
CREATE INDEX index_team_membership_events_on_team_id_and_created_at ON team_membership_events (team_id, created_at);
CREATE INDEX index_team_membership_events_on_team_id ON team_membership_events (team_id);
CREATE INDEX index_team_membership_events_on_user_id ON team_membership_events (user_id);

CREATE TRIGGER set_team_membership_events_updated_at
    BEFORE UPDATE ON team_membership_events
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- games
CREATE TABLE games (
    id bigserial PRIMARY KEY,
    name text,
    organizer text,
    starts_at timestamptz,
    ends_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    avatar_url text,
    site_url text,
    ctftime_url text,
    finalized boolean NOT NULL DEFAULT false,
    finalized_at timestamptz,
    registration_opens_at timestamptz,
    registration_closes_at timestamptz,
    scoreboard_opens_at timestamptz,
    scoreboard_closes_at timestamptz,
    vpn_url text,
    vpn_config_url text,
    access_instructions text,
    access_secret text
);

CREATE TRIGGER set_games_updated_at
    BEFORE UPDATE ON games
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- services
CREATE TABLE services (
    id bigserial PRIMARY KEY,
    name text NOT NULL,
    public_description text,
    private_description text,
    author text,
    copyright text,
    avatar_url text,
    public boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    service_archive_url text,
    checker_archive_url text,
    writeup_url text,
    exploits_url text,
    check_status text NOT NULL DEFAULT 'unknown',
    checked_at timestamptz,
    service_local_path text,
    service_local_size integer,
    service_local_sha256 text,
    service_downloaded_at timestamptz,
    checker_local_path text,
    checker_local_size integer,
    checker_local_sha256 text,
    checker_downloaded_at timestamptz,
    ctf01d_training jsonb NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX index_services_on_name ON services (name);

CREATE TRIGGER set_services_updated_at
    BEFORE UPDATE ON services
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- games_services (join table, no id)
CREATE TABLE games_services (
    game_id bigint NOT NULL,
    service_id bigint NOT NULL
);

CREATE UNIQUE INDEX index_games_services_on_game_id_and_service_id ON games_services (game_id, service_id);
CREATE INDEX index_games_services_on_service_id_and_game_id ON games_services (service_id, game_id);

-- game_teams
CREATE TABLE game_teams (
    id bigserial PRIMARY KEY,
    game_id bigint NOT NULL,
    team_id bigint NOT NULL,
    ip_address text,
    ctf01d_id text,
    ctf01d_overrides jsonb NOT NULL DEFAULT '{}',
    team_type text,
    "order" integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX index_game_teams_on_game_id_and_order_and_id ON game_teams (game_id, "order", id);
CREATE UNIQUE INDEX index_game_teams_on_game_id_and_team_id ON game_teams (game_id, team_id);
CREATE INDEX index_game_teams_on_game_id ON game_teams (game_id);
CREATE INDEX index_game_teams_on_team_id ON game_teams (team_id);

CREATE TRIGGER set_game_teams_updated_at
    BEFORE UPDATE ON game_teams
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- results
CREATE TABLE results (
    id bigserial PRIMARY KEY,
    game_id bigint NOT NULL,
    team_id bigint NOT NULL,
    score integer,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX index_results_on_game_id_and_team_id ON results (game_id, team_id);
CREATE INDEX index_results_on_game_id ON results (game_id);
CREATE INDEX index_results_on_team_id ON results (team_id);

CREATE TRIGGER set_results_updated_at
    BEFORE UPDATE ON results
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- final_results
CREATE TABLE final_results (
    id bigserial PRIMARY KEY,
    game_id bigint NOT NULL,
    team_id bigint NOT NULL,
    score integer NOT NULL DEFAULT 0,
    position integer,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX index_final_results_on_game_id_and_team_id ON final_results (game_id, team_id);
CREATE INDEX index_final_results_on_game_id ON final_results (game_id);
CREATE INDEX index_final_results_on_team_id ON final_results (team_id);

CREATE TRIGGER set_final_results_updated_at
    BEFORE UPDATE ON final_results
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- writeups
CREATE TABLE writeups (
    id bigserial PRIMARY KEY,
    game_id bigint NOT NULL,
    team_id bigint NOT NULL,
    title text NOT NULL,
    url text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX index_writeups_on_game_id_and_team_id_and_title ON writeups (game_id, team_id, title);
CREATE INDEX index_writeups_on_game_id ON writeups (game_id);
CREATE INDEX index_writeups_on_team_id ON writeups (team_id);

CREATE TRIGGER set_writeups_updated_at
    BEFORE UPDATE ON writeups
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Foreign keys
ALTER TABLE ONLY teams
    ADD CONSTRAINT fk_teams_university_id
    FOREIGN KEY (university_id) REFERENCES universities(id);

ALTER TABLE ONLY team_memberships
    ADD CONSTRAINT fk_team_memberships_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id);

ALTER TABLE ONLY team_memberships
    ADD CONSTRAINT fk_team_memberships_user_id
    FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE ONLY team_membership_events
    ADD CONSTRAINT fk_team_membership_events_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id);

ALTER TABLE ONLY team_membership_events
    ADD CONSTRAINT fk_team_membership_events_user_id
    FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE ONLY game_teams
    ADD CONSTRAINT fk_game_teams_game_id
    FOREIGN KEY (game_id) REFERENCES games(id);

ALTER TABLE ONLY game_teams
    ADD CONSTRAINT fk_game_teams_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id);

ALTER TABLE ONLY games_services
    ADD CONSTRAINT fk_games_services_game_id
    FOREIGN KEY (game_id) REFERENCES games(id);

ALTER TABLE ONLY games_services
    ADD CONSTRAINT fk_games_services_service_id
    FOREIGN KEY (service_id) REFERENCES services(id);

ALTER TABLE ONLY results
    ADD CONSTRAINT fk_results_game_id
    FOREIGN KEY (game_id) REFERENCES games(id);

ALTER TABLE ONLY results
    ADD CONSTRAINT fk_results_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id);

ALTER TABLE ONLY final_results
    ADD CONSTRAINT fk_final_results_game_id
    FOREIGN KEY (game_id) REFERENCES games(id);

ALTER TABLE ONLY final_results
    ADD CONSTRAINT fk_final_results_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id);

ALTER TABLE ONLY writeups
    ADD CONSTRAINT fk_writeups_game_id
    FOREIGN KEY (game_id) REFERENCES games(id);

ALTER TABLE ONLY writeups
    ADD CONSTRAINT fk_writeups_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id);

-- +goose Down
-- Reverse the migration: drop all objects in dependency order

DROP TRIGGER IF EXISTS set_writeups_updated_at ON writeups;
DROP TRIGGER IF EXISTS set_final_results_updated_at ON final_results;
DROP TRIGGER IF EXISTS set_results_updated_at ON results;
DROP TRIGGER IF EXISTS set_game_teams_updated_at ON game_teams;
DROP TRIGGER IF EXISTS set_services_updated_at ON services;
DROP TRIGGER IF EXISTS set_games_updated_at ON games;
DROP TRIGGER IF EXISTS set_team_membership_events_updated_at ON team_membership_events;
DROP TRIGGER IF EXISTS set_team_memberships_updated_at ON team_memberships;
DROP TRIGGER IF EXISTS set_teams_updated_at ON teams;
DROP TRIGGER IF EXISTS set_universities_updated_at ON universities;
DROP TRIGGER IF EXISTS set_users_updated_at ON users;

DROP TABLE IF EXISTS writeups;
DROP TABLE IF EXISTS final_results;
DROP TABLE IF EXISTS results;
DROP TABLE IF EXISTS game_teams;
DROP TABLE IF EXISTS games_services;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS games;
DROP TABLE IF EXISTS team_membership_events;
DROP TABLE IF EXISTS team_memberships;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS universities;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS set_updated_at();
