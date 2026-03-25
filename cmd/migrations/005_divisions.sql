-- =====================================
-- Table: divisions
-- Stores ELO thresholds for each division
-- =====================================
CREATE TABLE IF NOT EXISTS divisions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    min_elo INTEGER NOT NULL DEFAULT 0,
    max_elo INTEGER,
    category TEXT NOT NULL DEFAULT 'both',
    color TEXT NOT NULL DEFAULT '#ffffff',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT
);

-- Seed default divisions
INSERT OR IGNORE INTO divisions (id, name, display_order, min_elo, max_elo, category, color) VALUES
    ('div-champion', 'Champion', 1, 2000, NULL, 'both', '#FFD700'),
    ('div-first', 'First Division', 2, 1600, 1999, 'both', '#C0C0C0'),
    ('div-second', 'Second Division', 3, 1300, 1599, 'both', '#CD7F32'),
    ('div-third', 'Third Division', 4, 1000, 1299, 'both', '#4A90D9'),
    ('div-fourth', 'Fourth Division', 5, 0, 999, 'both', '#7B8794');
