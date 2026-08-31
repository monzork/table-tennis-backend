-- Lets each division scale the flat champion Elo placement bonus on its own
-- terms instead of every division sharing the same 2x-K-factor champion /
-- 1x runner-up / 0.5x third bonus (see event.PlacementEloBonus). Defaults to
-- 2, matching that exact prior flat behavior for every existing division.
ALTER TABLE divisions ADD COLUMN IF NOT EXISTS placement_elo_multiplier NUMERIC NOT NULL DEFAULT 2;

-- Seed the values requested for the existing gender-specific bands: the
-- elite division earns the smallest champion bonus since its players are
-- already top-rated, the lowest division the largest to reward climbing.
UPDATE divisions SET placement_elo_multiplier = 0.5 WHERE id IN ('div-first-male', 'div-first-female');
UPDATE divisions SET placement_elo_multiplier = 1   WHERE id IN ('div-second-male', 'div-second-female');
UPDATE divisions SET placement_elo_multiplier = 2   WHERE id = 'div-third-male';
