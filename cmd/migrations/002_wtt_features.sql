-- Migration for WTT Features

-- 1. Players Table
ALTER TABLE players ADD COLUMN gender TEXT NOT NULL DEFAULT 'M';
ALTER TABLE players ADD COLUMN singles_elo INTEGER NOT NULL DEFAULT 1000;
ALTER TABLE players ADD COLUMN doubles_elo INTEGER NOT NULL DEFAULT 1000;

-- Copy old elo to singles_elo
UPDATE players SET singles_elo = elo, doubles_elo = elo;

-- Drop old elo column (Requires SQLite 3.35.0+)
ALTER TABLE players DROP COLUMN elo;

-- 2. Tournaments Table
ALTER TABLE tournaments ADD COLUMN type TEXT NOT NULL DEFAULT 'singles';

-- 3. Matches Table (Recreating to handle foreign keys and drop constraints safely in SQLite)
CREATE TABLE matches_new (
    id TEXT PRIMARY KEY,
    tournament_id TEXT NOT NULL,
    match_type TEXT NOT NULL DEFAULT 'singles', -- 'singles' or 'doubles'
    team_a_player_1_id TEXT NOT NULL,
    team_a_player_2_id TEXT,
    team_b_player_1_id TEXT NOT NULL,
    team_b_player_2_id TEXT,
    status TEXT NOT NULL DEFAULT 'scheduled',
    winner_team TEXT, -- 'A', 'B', or NULL
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id),
    FOREIGN KEY (team_a_player_1_id) REFERENCES players(id),
    FOREIGN KEY (team_a_player_2_id) REFERENCES players(id),
    FOREIGN KEY (team_b_player_1_id) REFERENCES players(id),
    FOREIGN KEY (team_b_player_2_id) REFERENCES players(id)
);

INSERT INTO matches_new (
    id, tournament_id, match_type,
    team_a_player_1_id, team_b_player_1_id,
    status, created_at, updated_at
)
SELECT 
    id, tournament_id, 'singles',
    player_a_id, player_b_id,
    status, created_at, updated_at
FROM matches;

-- We don't have winner_team in the old one, we had winner_id. Let's update winner_team based on winner_id.
UPDATE matches_new SET winner_team = 'A' WHERE EXISTS (SELECT 1 FROM matches WHERE matches.id = matches_new.id AND matches.winner_id = matches_new.team_a_player_1_id);
UPDATE matches_new SET winner_team = 'B' WHERE EXISTS (SELECT 1 FROM matches WHERE matches.id = matches_new.id AND matches.winner_id = matches_new.team_b_player_1_id);

DROP TABLE matches;
ALTER TABLE matches_new RENAME TO matches;
