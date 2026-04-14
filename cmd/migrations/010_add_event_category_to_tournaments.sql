-- Migration 010: Add event_category to tournaments
-- Adds the event_category column to differentiate between men, women, mixed, open correctly.

ALTER TABLE tournaments ADD COLUMN event_category TEXT NOT NULL DEFAULT 'open';
