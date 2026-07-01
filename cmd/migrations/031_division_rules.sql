-- Division-specific rules per tournament
-- Allows different divisions to have different match formats (e.g., Best of 3 vs Best of 5)

CREATE TABLE IF NOT EXISTS tournament_division_rules (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    division_id VARCHAR(36) NOT NULL REFERENCES divisions(id) ON DELETE CASCADE,
    best_of INTEGER NOT NULL CHECK (best_of >= 3 AND best_of % 2 = 1),
    points_to_win INTEGER NOT NULL CHECK (points_to_win >= 1),
    points_margin INTEGER NOT NULL CHECK (points_margin >= 1),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tournament_id, division_id)
);

CREATE INDEX IF NOT EXISTS idx_tournament_division_rules_tournament 
ON tournament_division_rules(tournament_id);

CREATE INDEX IF NOT EXISTS idx_tournament_division_rules_division 
ON tournament_division_rules(division_id);