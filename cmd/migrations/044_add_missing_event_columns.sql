ALTER TABLE events ADD COLUMN IF NOT EXISTS losers_group_pass_count int NOT NULL DEFAULT 0;
ALTER TABLE events ADD COLUMN IF NOT EXISTS division_configs jsonb;
ALTER TABLE events ADD COLUMN IF NOT EXISTS manual_seeding_locked boolean NOT NULL DEFAULT false;
