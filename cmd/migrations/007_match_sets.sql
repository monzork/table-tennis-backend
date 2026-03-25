-- 007_match_sets.sql
-- Stores individual set scores for a match

CREATE TABLE match_sets (
    id          TEXT PRIMARY KEY,
    match_id    TEXT NOT NULL,
    set_number  INTEGER NOT NULL,
    score_a     INTEGER NOT NULL DEFAULT 0,
    score_b     INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (match_id) REFERENCES matches(id) ON DELETE CASCADE,
    UNIQUE(match_id, set_number)
);
