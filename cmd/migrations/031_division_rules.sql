-- Division-specific rules per event
-- Allows different divisions to have different match formats (e.g., Best of 3 vs Best of 5)

CREATE TABLE IF NOT EXISTS event_division_rules (
    id VARCHAR(36) PRIMARY KEY,
    event_id VARCHAR(36) NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    division_id VARCHAR(36) NOT NULL REFERENCES divisions(id) ON DELETE CASCADE,
    best_of INTEGER NOT NULL CHECK (best_of >= 3 AND best_of % 2 = 1),
    points_to_win INTEGER NOT NULL CHECK (points_to_win >= 1),
    points_margin INTEGER NOT NULL CHECK (points_margin >= 1),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, division_id)
);

CREATE INDEX IF NOT EXISTS idx_event_division_rules_event 
ON event_division_rules(event_id);

CREATE INDEX IF NOT EXISTS idx_event_division_rules_division 
ON event_division_rules(division_id);