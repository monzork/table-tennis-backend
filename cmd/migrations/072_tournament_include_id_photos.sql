-- Lets each tournament opt in to appending player cedula de identidad
-- photos to its exported PDF report -- off by default, since downloading
-- and embedding every player's ID photo is the slowest/most memory-heavy
-- part of report generation.
ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS include_id_photos_in_report BOOLEAN NOT NULL DEFAULT FALSE;
