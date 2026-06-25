-- Migration 011: Add second_name and second_last_name to players table
-- Adds the second_name and second_last_name columns to accommodate middle names and maternal surnames.

ALTER TABLE players ADD COLUMN second_name TEXT NOT NULL DEFAULT '';
ALTER TABLE players ADD COLUMN second_last_name TEXT NOT NULL DEFAULT '';
