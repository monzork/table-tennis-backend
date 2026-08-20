-- Lets a division define its Elo band per gender (e.g. separate men's/women's
-- 1st Division thresholds) now that singles Elo is one shared pool.
ALTER TABLE divisions ADD COLUMN IF NOT EXISTS gender TEXT NOT NULL DEFAULT 'both';
