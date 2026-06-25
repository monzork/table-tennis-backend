-- Migration 024: Tournament-specific PINs
-- Drop pin column from players (PINs are now per-tournament-per-player)
ALTER TABLE players DROP COLUMN IF EXISTS pin;

-- Add pin column to tournament_participants (4-digit PIN per player per tournament)
ALTER TABLE tournament_participants ADD COLUMN IF NOT EXISTS pin TEXT NOT NULL DEFAULT '0000';

-- Generate random 4-digit PINs for existing participants
UPDATE tournament_participants
SET pin = LPAD(FLOOR(RANDOM() * 10000)::TEXT, 4, '0')
WHERE pin = '0000';

-- Add num_tables to events if not present (should already exist from migration 019, but be safe)
ALTER TABLE events ADD COLUMN IF NOT EXISTS num_tables INT NOT NULL DEFAULT 4;

-- Add num_tables to tournaments if not present
ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS num_tables INT NOT NULL DEFAULT 0;
