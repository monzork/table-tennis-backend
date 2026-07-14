-- Add stage column to division rules to support stage-specific division rules
-- This allows different divisions to have different rules per stage (e.g., 1st Division Best of 5 in finals, 2nd Division Best of 3 in finals)

ALTER TABLE event_division_rules 
ADD COLUMN IF NOT EXISTS stage VARCHAR(50) DEFAULT 'group';

-- Drop the old unique constraint on (event_id, division_id)
-- We'll add a new one that includes stage
DROP INDEX IF EXISTS idx_event_division_rules_unique;

-- Add new unique constraint that includes stage
CREATE UNIQUE INDEX IF NOT EXISTS idx_event_division_rules_unique 
ON event_division_rules(event_id, division_id, stage);

-- Update existing records to have 'group' stage (default)
UPDATE event_division_rules 
SET stage = 'group' 
WHERE stage IS NULL;