-- Add stage column to division rules to support stage-specific division rules
-- This allows different divisions to have different rules per stage (e.g., 1st Division Best of 5 in finals, 2nd Division Best of 3 in finals)

ALTER TABLE tournament_division_rules 
ADD COLUMN IF NOT EXISTS stage VARCHAR(50) DEFAULT 'group';

-- Drop the old unique constraint on (tournament_id, division_id)
-- We'll add a new one that includes stage
DROP INDEX IF EXISTS idx_tournament_division_rules_unique;

-- Add new unique constraint that includes stage
CREATE UNIQUE INDEX IF NOT EXISTS idx_tournament_division_rules_unique 
ON tournament_division_rules(tournament_id, division_id, stage);

-- Update existing records to have 'group' stage (default)
UPDATE tournament_division_rules 
SET stage = 'group' 
WHERE stage IS NULL;