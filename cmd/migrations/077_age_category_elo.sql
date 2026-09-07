-- Age-category Elo: U-11/U-13/U-15/U-19 youth brackets each carry their own
-- Singles/Doubles rating, fully separate from the existing (Open/adult)
-- singles_elo/doubles_elo columns. events.age_category tags which pool a
-- given event's matches read/write ("open" is the default for every
-- existing row and every event created without an explicit age tier).
ALTER TABLE players ADD COLUMN IF NOT EXISTS singles_elo_u11 SMALLINT NOT NULL DEFAULT 1000;
ALTER TABLE players ADD COLUMN IF NOT EXISTS doubles_elo_u11 SMALLINT NOT NULL DEFAULT 1000;
ALTER TABLE players ADD COLUMN IF NOT EXISTS singles_elo_u13 SMALLINT NOT NULL DEFAULT 1000;
ALTER TABLE players ADD COLUMN IF NOT EXISTS doubles_elo_u13 SMALLINT NOT NULL DEFAULT 1000;
ALTER TABLE players ADD COLUMN IF NOT EXISTS singles_elo_u15 SMALLINT NOT NULL DEFAULT 1000;
ALTER TABLE players ADD COLUMN IF NOT EXISTS doubles_elo_u15 SMALLINT NOT NULL DEFAULT 1000;
ALTER TABLE players ADD COLUMN IF NOT EXISTS singles_elo_u19 SMALLINT NOT NULL DEFAULT 1000;
ALTER TABLE players ADD COLUMN IF NOT EXISTS doubles_elo_u19 SMALLINT NOT NULL DEFAULT 1000;

ALTER TABLE events ADD COLUMN IF NOT EXISTS age_category TEXT NOT NULL DEFAULT 'open';
