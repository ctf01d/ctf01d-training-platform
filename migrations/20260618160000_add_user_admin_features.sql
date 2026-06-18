-- +goose Up
-- Admin user management: optional profile fields, blocking, login tracking and
-- persisted sessions so admins can inspect/revoke active logins.

ALTER TABLE users
    ADD COLUMN bio text,
    ADD COLUMN telegram text,
    ADD COLUMN github text,
    ADD COLUMN email text,
    ADD COLUMN is_blocked boolean NOT NULL DEFAULT false,
    ADD COLUMN blocked_at timestamptz,
    ADD COLUMN last_login_ip text,
    ADD COLUMN last_login_at timestamptz;

CREATE TABLE user_sessions (
    id bigserial PRIMARY KEY,
    user_id bigint NOT NULL,
    jti text NOT NULL,
    ip_address text,
    user_agent text,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz
);

CREATE UNIQUE INDEX index_user_sessions_on_jti ON user_sessions (jti);
CREATE INDEX index_user_sessions_on_user_id ON user_sessions (user_id);

ALTER TABLE ONLY user_sessions
    ADD CONSTRAINT fk_user_sessions_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Removing a user must also remove their team data and references.
ALTER TABLE ONLY team_memberships
    DROP CONSTRAINT fk_team_memberships_user_id,
    ADD CONSTRAINT fk_team_memberships_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY team_membership_events
    DROP CONSTRAINT fk_team_membership_events_user_id,
    ADD CONSTRAINT fk_team_membership_events_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- +goose Down

ALTER TABLE ONLY team_membership_events
    DROP CONSTRAINT fk_team_membership_events_user_id,
    ADD CONSTRAINT fk_team_membership_events_user_id
    FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE ONLY team_memberships
    DROP CONSTRAINT fk_team_memberships_user_id,
    ADD CONSTRAINT fk_team_memberships_user_id
    FOREIGN KEY (user_id) REFERENCES users(id);

DROP TABLE IF EXISTS user_sessions;

ALTER TABLE users
    DROP COLUMN bio,
    DROP COLUMN telegram,
    DROP COLUMN github,
    DROP COLUMN email,
    DROP COLUMN is_blocked,
    DROP COLUMN blocked_at,
    DROP COLUMN last_login_ip,
    DROP COLUMN last_login_at;
