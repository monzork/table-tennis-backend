ALTER TABLE tournaments ADD COLUMN division_group_counts JSONB NOT NULL DEFAULT '{}'::jsonb;
