CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY,
    google_sub TEXT UNIQUE NOT NULL,
    email TEXT NOT NULL,
    name TEXT,
    picture_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);
ALTER TABLE players ADD COLUMN IF NOT EXISTS guardian_account_id UUID REFERENCES accounts(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_players_guardian_account_id ON players(guardian_account_id);
