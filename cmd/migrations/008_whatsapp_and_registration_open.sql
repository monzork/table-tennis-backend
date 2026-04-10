-- Migration: Add whatsapp_number to players and registration_open to tournaments
-- Date: 2026-04-10

-- Add whatsapp_number to players table
ALTER TABLE players ADD COLUMN whatsapp_number TEXT;

-- Add registration_open to tournaments table
ALTER TABLE tournaments ADD COLUMN registration_open BOOLEAN NOT NULL DEFAULT 0;
