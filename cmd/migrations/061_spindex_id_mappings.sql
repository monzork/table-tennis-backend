-- Adds Spindex external-ID columns so the Spindex data import script
-- (cmd/import_spindex.go) can resolve entities on first run and be
-- idempotent on every re-run (only newly-completed Spindex matches get
-- processed each sync).
ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS spindex_tournament_id TEXT UNIQUE;
ALTER TABLE events ADD COLUMN IF NOT EXISTS spindex_event_id TEXT UNIQUE;
ALTER TABLE players ADD COLUMN IF NOT EXISTS spindex_player_id TEXT UNIQUE;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS spindex_match_id TEXT UNIQUE;
