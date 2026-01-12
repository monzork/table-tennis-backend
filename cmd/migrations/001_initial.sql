-- =====================================
-- Table: players
-- =====================================
CREATE TABLE IF NOT EXISTS players (
    id TEXT PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    birthdate TEXT NOT NULL,
    elo INTEGER NOT NULL DEFAULT 1000,
    country TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT
);

-- =====================================
-- Table: tournaments
-- =====================================
CREATE TABLE IF NOT EXISTS tournaments (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT
);

-- =====================================
-- Table: rules
-- =====================================
CREATE TABLE IF NOT EXISTS rules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT
);

-- =====================================
-- Table: matches
-- =====================================
CREATE TABLE IF NOT EXISTS matches (
    id TEXT PRIMARY KEY,
    tournament_id TEXT NOT NULL,
    player_a_id TEXT NOT NULL,
    player_b_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'scheduled',
    winner_id TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id),
    FOREIGN KEY (player_a_id) REFERENCES players(id),
    FOREIGN KEY (player_b_id) REFERENCES players(id),
    FOREIGN KEY (winner_id) REFERENCES players(id)
);
