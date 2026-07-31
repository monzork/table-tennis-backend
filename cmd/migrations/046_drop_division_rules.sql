-- Migration 046: per-event division-specific rules/config are redundant now
-- that events are created one-per-division by the standard tournament flow;
-- the standalone single-event flow only needs a single-bucket seeder.
DROP TABLE IF EXISTS event_division_rules;
ALTER TABLE events DROP COLUMN IF EXISTS division_configs;
