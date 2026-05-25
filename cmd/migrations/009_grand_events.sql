-- =====================================
-- Table: events
-- =====================================
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    division_id TEXT NOT NULL REFERENCES divisions(id),
    skip_elo BOOLEAN NOT NULL DEFAULT FALSE,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);
-- Add event_id and skip_elo to tournaments table
ALTER TABLE tournaments
ADD COLUMN IF NOT EXISTS event_id UUID REFERENCES events(id) ON DELETE
SET NULL;
ALTER TABLE tournaments
ADD COLUMN IF NOT EXISTS skip_elo BOOLEAN NOT NULL DEFAULT FALSE;