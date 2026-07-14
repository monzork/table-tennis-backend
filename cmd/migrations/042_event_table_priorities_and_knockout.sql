-- Migration: add table priorities to tournaments and knockout_stage_started to events
-- This assumes 041_swap_event_and_tournament.sql has run, so naming reflects the new swapped logic!

-- Add properties to the new 'tournaments' table (top-level entity)
ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS table_priorities JSONB;

-- Add properties to the new 'events' table (sub-entity)
ALTER TABLE events ADD COLUMN IF NOT EXISTS knockout_stage_started BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE events ADD COLUMN IF NOT EXISTS manual_seeding_locked BOOLEAN NOT NULL DEFAULT false;
