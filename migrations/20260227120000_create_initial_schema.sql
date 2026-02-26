-- +goose Up

-- Users (ADR-005)
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(50)  NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Refresh Tokens (ADR-005)
CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64)  NOT NULL,
    expires_at TIMESTAMPTZ  NOT NULL,
    revoked    BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);

-- Scenarios (ADR-003)
CREATE TABLE scenarios (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id   UUID         NOT NULL REFERENCES users(id),
    title       VARCHAR(200) NOT NULL,
    description TEXT,
    version     INT          NOT NULL DEFAULT 1,
    status      VARCHAR(20)  NOT NULL DEFAULT 'draft',
    content     JSONB        NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_scenarios_author ON scenarios(author_id);
CREATE INDEX idx_scenarios_status ON scenarios(status);

-- Characters (architecture.md ER diagram)
CREATE TABLE characters (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES users(id),
    name       VARCHAR(100) NOT NULL,
    attributes JSONB        DEFAULT '{}',
    inventory  JSONB        DEFAULT '[]',
    notes      TEXT         DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_characters_user ON characters(user_id);

-- Game Sessions (ADR-004 + ADR-003 notes)
CREATE TABLE game_sessions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id  UUID        NOT NULL REFERENCES scenarios(id),
    gm_id        UUID        NOT NULL REFERENCES users(id),
    status       VARCHAR(20) NOT NULL DEFAULT 'lobby',
    invite_code  VARCHAR(10) NOT NULL UNIQUE,
    state        JSONB       DEFAULT '{}',
    gm_notes     JSONB       DEFAULT '{}',
    snapshot_seq BIGINT      DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at   TIMESTAMPTZ,
    ended_at     TIMESTAMPTZ
);
CREATE INDEX idx_game_sessions_gm     ON game_sessions(gm_id);
CREATE INDEX idx_game_sessions_status  ON game_sessions(status);
CREATE INDEX idx_game_sessions_invite  ON game_sessions(invite_code);

-- Session Players (ADR-004 + ADR-003 notes)
CREATE TABLE session_players (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id    UUID        NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    user_id       UUID        NOT NULL REFERENCES users(id),
    character_id  UUID        REFERENCES characters(id),
    current_scene VARCHAR(100),
    status        VARCHAR(20) NOT NULL DEFAULT 'joined',
    notes         TEXT        DEFAULT '',
    joined_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, user_id)
);
CREATE INDEX idx_session_players_session ON session_players(session_id);

-- Game Events (ADR-004)
CREATE TABLE game_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID        NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    sequence   BIGINT      NOT NULL,
    type       VARCHAR(50) NOT NULL,
    actor_id   UUID        REFERENCES users(id),
    payload    JSONB       NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, sequence)
);
CREATE INDEX idx_game_events_session_seq ON game_events(session_id, sequence);

-- Revealed Items (ADR-004 + architecture.md)
CREATE TABLE revealed_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID         NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    player_id   UUID         NOT NULL REFERENCES users(id),
    item_id     VARCHAR(100) NOT NULL,
    revealed_by VARCHAR(20)  NOT NULL DEFAULT 'gm_manual',
    revealed_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, player_id, item_id)
);
CREATE INDEX idx_revealed_items_session ON revealed_items(session_id);

-- Revealed NPC Fields (ADR-003 + ADR-004)
CREATE TABLE revealed_npc_fields (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID         NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    player_id   UUID         NOT NULL REFERENCES users(id),
    npc_id      VARCHAR(100) NOT NULL,
    field_key   VARCHAR(100) NOT NULL,
    revealed_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, player_id, npc_id, field_key)
);
CREATE INDEX idx_revealed_npc_fields_session ON revealed_npc_fields(session_id);

-- +goose Down

DROP TABLE IF EXISTS revealed_npc_fields;
DROP TABLE IF EXISTS revealed_items;
DROP TABLE IF EXISTS game_events;
DROP TABLE IF EXISTS session_players;
DROP TABLE IF EXISTS game_sessions;
DROP TABLE IF EXISTS characters;
DROP TABLE IF EXISTS scenarios;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
