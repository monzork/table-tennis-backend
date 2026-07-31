-- Migration 045: events.tournament_category actually describes the event's own
-- category (men/women/mixed/open), not the parent tournament's — rename it to
-- match what it holds.
ALTER TABLE events RENAME COLUMN tournament_category TO event_category;
