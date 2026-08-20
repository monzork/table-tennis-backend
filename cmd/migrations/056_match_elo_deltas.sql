-- Per-match Elo points gained/lost by each team, so a player's match
-- history can show how many points a specific result was worth.
ALTER TABLE matches ADD COLUMN IF NOT EXISTS elo_delta_a DOUBLE PRECISION;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS elo_delta_b DOUBLE PRECISION;
