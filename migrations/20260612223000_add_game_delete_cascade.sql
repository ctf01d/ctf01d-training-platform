-- +goose Up
-- Delete game-owned data automatically when a game is removed.

ALTER TABLE ONLY game_teams
    DROP CONSTRAINT fk_game_teams_game_id,
    ADD CONSTRAINT fk_game_teams_game_id
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE;

ALTER TABLE ONLY games_services
    DROP CONSTRAINT fk_games_services_game_id,
    ADD CONSTRAINT fk_games_services_game_id
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE;

ALTER TABLE ONLY results
    DROP CONSTRAINT fk_results_game_id,
    ADD CONSTRAINT fk_results_game_id
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE;

ALTER TABLE ONLY final_results
    DROP CONSTRAINT fk_final_results_game_id,
    ADD CONSTRAINT fk_final_results_game_id
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE;

ALTER TABLE ONLY writeups
    DROP CONSTRAINT fk_writeups_game_id,
    ADD CONSTRAINT fk_writeups_game_id
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE;

-- +goose Down

ALTER TABLE ONLY writeups
    DROP CONSTRAINT fk_writeups_game_id,
    ADD CONSTRAINT fk_writeups_game_id
    FOREIGN KEY (game_id) REFERENCES games(id);

ALTER TABLE ONLY final_results
    DROP CONSTRAINT fk_final_results_game_id,
    ADD CONSTRAINT fk_final_results_game_id
    FOREIGN KEY (game_id) REFERENCES games(id);

ALTER TABLE ONLY results
    DROP CONSTRAINT fk_results_game_id,
    ADD CONSTRAINT fk_results_game_id
    FOREIGN KEY (game_id) REFERENCES games(id);

ALTER TABLE ONLY games_services
    DROP CONSTRAINT fk_games_services_game_id,
    ADD CONSTRAINT fk_games_services_game_id
    FOREIGN KEY (game_id) REFERENCES games(id);

ALTER TABLE ONLY game_teams
    DROP CONSTRAINT fk_game_teams_game_id,
    ADD CONSTRAINT fk_game_teams_game_id
    FOREIGN KEY (game_id) REFERENCES games(id);
