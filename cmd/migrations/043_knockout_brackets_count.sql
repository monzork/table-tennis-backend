-- up
ALTER TABLE events ADD COLUMN knockout_brackets_count INTEGER NOT NULL DEFAULT 1;

-- down
ALTER TABLE events DROP COLUMN knockout_brackets_count;
