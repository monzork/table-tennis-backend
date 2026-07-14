-- Migration: Swap 'tournaments' and 'events' naming
-- This is a pure renaming migration to preserve data.

-- 1. Swap main tables
ALTER TABLE events RENAME TO events_tmp;
ALTER TABLE tournaments RENAME TO events;
ALTER TABLE events_tmp RENAME TO tournaments;

-- 2. Rename 'event_id' to 'tournament_id' and 'event_category' to 'tournament_category' in the new 'events' table
ALTER TABLE events RENAME COLUMN event_id TO tournament_id;
ALTER TABLE events RENAME COLUMN event_category TO tournament_category;

-- 3. Rename 'tournament_id' to 'event_id' in all other tables
ALTER TABLE groups RENAME COLUMN tournament_id TO event_id;
ALTER TABLE matches RENAME COLUMN tournament_id TO event_id;
ALTER TABLE tournament_participants RENAME COLUMN tournament_id TO event_id;
ALTER TABLE tournament_stage_rules RENAME COLUMN tournament_id TO event_id;

-- 4. Rename tables that had 'tournament' in their name
ALTER TABLE tournament_participants RENAME TO event_participants;
ALTER TABLE tournament_stage_rules RENAME TO event_stage_rules;
