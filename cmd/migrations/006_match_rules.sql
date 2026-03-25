-- 006_match_rules.sql

-- 1. Per-stage match rules per tournament
CREATE TABLE tournament_stage_rules (
    id                TEXT PRIMARY KEY,
    tournament_id     TEXT NOT NULL,
    stage             TEXT NOT NULL,   -- 'group','r32','r16','quarterfinal','semifinal','final'
    best_of           INTEGER NOT NULL DEFAULT 5,
    points_to_win     INTEGER NOT NULL DEFAULT 11,
    points_margin     INTEGER NOT NULL DEFAULT 2,
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    UNIQUE(tournament_id, stage)
);

-- 2. Extend matches table with bracket context
ALTER TABLE matches ADD COLUMN stage TEXT NOT NULL DEFAULT 'group';
ALTER TABLE matches ADD COLUMN round_number INTEGER NOT NULL DEFAULT 1;
ALTER TABLE matches ADD COLUMN group_id TEXT REFERENCES groups(id);
ALTER TABLE matches ADD COLUMN next_match_id TEXT REFERENCES matches(id);
ALTER TABLE matches ADD COLUMN next_match_slot TEXT DEFAULT 'A'; -- 'A' or 'B' — which team slot to place the winner
