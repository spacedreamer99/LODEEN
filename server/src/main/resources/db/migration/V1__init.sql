CREATE TABLE IF NOT EXISTS players (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    registered_at INTEGER NOT NULL,
    last_seen     INTEGER NOT NULL,
    playtime_sec  INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_players_name ON players(name);
