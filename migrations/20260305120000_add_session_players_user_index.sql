-- +goose Up
CREATE INDEX idx_session_players_user ON session_players(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_session_players_user;
