-- Distinguishes a walkover (one side defaulted, scored as a normal
-- 11-0/11-0[/11-0] win via the Forfeit A/B button) from a genuinely played
-- match, so the UI can show a "Walkover" badge instead of a plain score
-- line. Doesn't change status/winner/Elo semantics -- see
-- internal/application/match/update_score.go.
ALTER TABLE matches ADD COLUMN IF NOT EXISTS is_forfeit BOOLEAN NOT NULL DEFAULT FALSE;

-- Backfill: "II Ranking Nacional por Divisiones" was played before this flag
-- existed, so every one of its finished matches whose sets are exactly
-- 11-0/11-0 (Bo3) or 11-0/11-0/11-0 (Bo5), in either direction, is almost
-- certainly a walkover recorded the old way rather than a genuine
-- blowout -- flag them as forfeits retroactively, and strip the fabricated
-- sets themselves (a forfeit going forward persists none at all, see
-- MatchRepository.UpdateScore) so they stop inflating sets/points stats.
CREATE TEMP TABLE forfeit_backfill AS
SELECT ms.match_id
FROM match_sets ms
JOIN matches mm ON mm.id = ms.match_id
JOIN events e ON e.id = mm.event_id
WHERE e.tournament_id = 'b800b28c-cd29-4585-b7bc-caa14abaf774'
  AND mm.status = 'finished'
GROUP BY ms.match_id
HAVING COUNT(*) IN (2, 3)
   AND bool_and((ms.score_a = 11 AND ms.score_b = 0) OR (ms.score_a = 0 AND ms.score_b = 11));

UPDATE matches
SET is_forfeit = TRUE
WHERE id IN (SELECT match_id FROM forfeit_backfill);

DELETE FROM match_sets
WHERE match_id IN (SELECT match_id FROM forfeit_backfill);

DROP TABLE forfeit_backfill;
