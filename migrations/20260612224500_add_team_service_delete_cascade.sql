-- +goose Up
-- Delete team/service-owned links automatically when parent rows are removed.

ALTER TABLE ONLY team_memberships
    DROP CONSTRAINT fk_team_memberships_team_id,
    ADD CONSTRAINT fk_team_memberships_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;

ALTER TABLE ONLY team_membership_events
    DROP CONSTRAINT fk_team_membership_events_team_id,
    ADD CONSTRAINT fk_team_membership_events_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;

ALTER TABLE ONLY game_teams
    DROP CONSTRAINT fk_game_teams_team_id,
    ADD CONSTRAINT fk_game_teams_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;

ALTER TABLE ONLY results
    DROP CONSTRAINT fk_results_team_id,
    ADD CONSTRAINT fk_results_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;

ALTER TABLE ONLY final_results
    DROP CONSTRAINT fk_final_results_team_id,
    ADD CONSTRAINT fk_final_results_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;

ALTER TABLE ONLY writeups
    DROP CONSTRAINT fk_writeups_team_id,
    ADD CONSTRAINT fk_writeups_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;

ALTER TABLE ONLY games_services
    DROP CONSTRAINT fk_games_services_service_id,
    ADD CONSTRAINT fk_games_services_service_id
    FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE;

-- +goose Down

ALTER TABLE ONLY games_services
    DROP CONSTRAINT fk_games_services_service_id,
    ADD CONSTRAINT fk_games_services_service_id
    FOREIGN KEY (service_id) REFERENCES services(id);

ALTER TABLE ONLY writeups
    DROP CONSTRAINT fk_writeups_team_id,
    ADD CONSTRAINT fk_writeups_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id);

ALTER TABLE ONLY final_results
    DROP CONSTRAINT fk_final_results_team_id,
    ADD CONSTRAINT fk_final_results_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id);

ALTER TABLE ONLY results
    DROP CONSTRAINT fk_results_team_id,
    ADD CONSTRAINT fk_results_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id);

ALTER TABLE ONLY game_teams
    DROP CONSTRAINT fk_game_teams_team_id,
    ADD CONSTRAINT fk_game_teams_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id);

ALTER TABLE ONLY team_membership_events
    DROP CONSTRAINT fk_team_membership_events_team_id,
    ADD CONSTRAINT fk_team_membership_events_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id);

ALTER TABLE ONLY team_memberships
    DROP CONSTRAINT fk_team_memberships_team_id,
    ADD CONSTRAINT fk_team_memberships_team_id
    FOREIGN KEY (team_id) REFERENCES teams(id);
