ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS division_formats JSON DEFAULT '{}';
