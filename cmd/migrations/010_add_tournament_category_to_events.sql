-- Migration 010: Add tournament_category to events
-- Adds the tournament_category column to differentiate between men, women, mixed, open correctly.

ALTER TABLE events ADD COLUMN tournament_category TEXT NOT NULL DEFAULT 'open';
