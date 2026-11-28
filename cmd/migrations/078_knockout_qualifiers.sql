-- Best next-placed qualifiers from the group stage, and optionally filling the
-- knockout bracket up to a chosen size with them. 0 keeps today's behaviour.
ALTER TABLE events ADD COLUMN IF NOT EXISTS best_extra_qualifiers INT NOT NULL DEFAULT 0;
ALTER TABLE events ADD COLUMN IF NOT EXISTS knockout_fill_size INT NOT NULL DEFAULT 0;
