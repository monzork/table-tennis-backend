ALTER TABLE events ADD COLUMN division_group_pass_counts JSONB NOT NULL DEFAULT '{}'::jsonb;
