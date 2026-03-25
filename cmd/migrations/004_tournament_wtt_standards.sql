-- 004_tournament_wtt_standards.sql

-- 1. Add Format to Tournaments
ALTER TABLE tournaments ADD COLUMN format TEXT NOT NULL DEFAULT 'elimination';

-- 2. Tournament Participants Table (M2M)
CREATE TABLE tournament_participants (
    tournament_id TEXT NOT NULL,
    player_id TEXT NOT NULL,
    PRIMARY KEY (tournament_id, player_id),
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
);

-- 3. Groups Table
CREATE TABLE groups (
    id TEXT PRIMARY KEY,
    tournament_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE
);

-- 4. Group Participants Table (M2M)
CREATE TABLE group_participants (
    group_id TEXT NOT NULL,
    player_id TEXT NOT NULL,
    PRIMARY KEY (group_id, player_id),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
);
