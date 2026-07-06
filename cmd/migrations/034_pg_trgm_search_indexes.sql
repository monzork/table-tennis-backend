-- Setup trigram indexes for fast full-text search on player names
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_player_fname_trgm ON players USING gin (LOWER(first_name) gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_player_sname_trgm ON players USING gin (LOWER(second_name) gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_player_lname_trgm ON players USING gin (LOWER(last_name) gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_player_slname_trgm ON players USING gin (LOWER(second_last_name) gin_trgm_ops);
