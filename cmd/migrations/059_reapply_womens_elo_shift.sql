-- Finishing the Orlando Jose in Memoriam - Women's Singles event
-- (23906d0c-8dc3-427d-921d-4a85099b114a) recalculated Elo from historical
-- event_participants snapshots captured before migration 051's -700
-- normalization, overwriting the singles Elo of every one of its 28
-- participants back to the pre-shift scale. Reapply the shift to exactly
-- those players (not all women -- the other female players were never
-- touched and are already correctly shifted).
UPDATE players
SET singles_elo = GREATEST(singles_elo - 700, 100)
WHERE gender = 'F'
  AND id IN (
    SELECT player_id FROM event_participants
    WHERE event_id = '23906d0c-8dc3-427d-921d-4a85099b114a'
  );

-- Fix the root cause so this can't recur: every event_participants snapshot
-- for a female player still holds its pre-migration-051 value (all of them
-- predate today's normalization). FinishTournamentUseCase/
-- RecalculateTournamentEloUseCase trust elo_before_singles as ground truth,
-- so any future finish/recalculate on ANY event with female participants
-- would silently re-corrupt their current Elo the same way. Shift every
-- stored singles snapshot for women by -700 to match.
UPDATE event_participants ep
SET elo_before_singles = GREATEST(ep.elo_before_singles - 700, 100)
FROM players p
WHERE p.id = ep.player_id AND p.gender = 'F' AND ep.elo_before_singles IS NOT NULL;

UPDATE event_participants ep
SET elo_after_singles = GREATEST(ep.elo_after_singles - 700, 100)
FROM players p
WHERE p.id = ep.player_id AND p.gender = 'F' AND ep.elo_after_singles IS NOT NULL;
