-- The inactivity Elo floor is no longer a single admin-configured rating
-- (migration 073's elo_floor) -- it's computed per player from their own
-- rating band (see inactivity.BandFloor: a 2101-rated player floors at
-- 2000, a 1859-rated one at 1700, etc). Each rating's computed floor is
-- fixed the moment it first goes inactive, so it doesn't keep sliding down
-- band by band on every later missed tournament.
ALTER TABLE inactivity_settings DROP COLUMN IF EXISTS elo_floor;

ALTER TABLE players ADD COLUMN IF NOT EXISTS inactivity_floor_singles SMALLINT;
ALTER TABLE players ADD COLUMN IF NOT EXISTS inactivity_floor_doubles SMALLINT;
