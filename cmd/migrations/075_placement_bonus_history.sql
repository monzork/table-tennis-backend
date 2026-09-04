-- Durable per-(event, player) record of a podium finish and the flat Elo
-- bonus it earned, written once when the event finishes and never touched
-- again by any later tournament -- so a player's placement-bonus history is
-- always queryable, not just recomputed live for the latest tournament.
-- See internal/domain/event.PlacementRecord.
ALTER TABLE event_participants ADD COLUMN IF NOT EXISTS placement TEXT;
ALTER TABLE event_participants ADD COLUMN IF NOT EXISTS placement_bonus_elo DOUBLE PRECISION;
