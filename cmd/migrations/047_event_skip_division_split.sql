ALTER TABLE events ADD COLUMN IF NOT EXISTS skip_division_split boolean NOT NULL DEFAULT false;
