-- Tracks how many Elo points a player's rating has actually shed to
-- inactivity decay during the current missed-tournament streak, so the
-- damage is visible (e.g. on the public ranking) rather than only knowing
-- the floor it's heading toward. Reset to 0 alongside the rest of the
-- streak the moment the player enrolls in a tournament again.
ALTER TABLE players ADD COLUMN IF NOT EXISTS lost_to_inactivity_singles SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS lost_to_inactivity_doubles SMALLINT NOT NULL DEFAULT 0;
