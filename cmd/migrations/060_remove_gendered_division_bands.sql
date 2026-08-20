-- Now that singles Elo is one unified pool for every player (migrations
-- 051/059), there's no more reason for divisions to have separate
-- male/female Elo thresholds -- Primera/Segunda/Tercera Division (already
-- reverted to Gender='both' in migration 058) apply the same band to
-- everyone. Remove the now-redundant per-gender band rows added in
-- migration 053.
DELETE FROM divisions WHERE id IN ('div-first-f', 'div-second-f', 'div-third-f');
