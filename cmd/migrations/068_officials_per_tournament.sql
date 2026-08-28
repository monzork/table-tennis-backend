-- Officials used to be scoped per child event. Move them to be scoped per
-- parent tournament (shared across all sibling events/categories), falling
-- back to the event's own ID when it has no parent tournament.

-- Dedupe first: two events under the same tournament could both have added
-- the same player as an official, which would collide once they share a key.
DELETE FROM event_officials eo
USING events e
WHERE eo.event_id = e.id
  AND EXISTS (
      SELECT 1 FROM event_officials eo2
      JOIN events e2 ON eo2.event_id = e2.id
      WHERE eo2.player_id = eo.player_id
        AND COALESCE(e2.tournament_id, e2.id) = COALESCE(e.tournament_id, e.id)
        AND eo2.event_id > eo.event_id
  );

ALTER TABLE event_officials RENAME COLUMN event_id TO tournament_id;

UPDATE event_officials eo
SET tournament_id = COALESCE(e.tournament_id, e.id)
FROM events e
WHERE eo.tournament_id = e.id;
