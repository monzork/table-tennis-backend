ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS federation_endorsed boolean NOT NULL DEFAULT false;
