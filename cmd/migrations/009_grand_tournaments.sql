-- =====================================
-- Table: tournaments
-- =====================================
CREATE TABLE IF NOT EXISTS tournaments (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    division_id TEXT NOT NULL REFERENCES divisions(id),
    skip_elo BOOLEAN NOT NULL DEFAULT FALSE,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);
-- Add tournament_id and skip_elo to events table
ALTER TABLE events
ADD COLUMN IF NOT EXISTS tournament_id UUID REFERENCES tournaments(id) ON DELETE
SET NULL;
ALTER TABLE events
ADD COLUMN IF NOT EXISTS skip_elo BOOLEAN NOT NULL DEFAULT FALSE;