ALTER TABLE tournaments ADD COLUMN division_group_pass_counts JSONB NOT NULL DEFAULT '{}'::jsonb;
