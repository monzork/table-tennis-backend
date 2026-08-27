ALTER TABLE players ADD COLUMN IF NOT EXISTS claimed_by_account_id UUID REFERENCES accounts(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_players_claimed_by_account_id ON players(claimed_by_account_id);
