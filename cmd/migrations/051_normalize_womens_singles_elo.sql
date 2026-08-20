-- One-time normalization: shift existing women's singles Elo down by 700
-- so men and women compete on a single unified singles Elo pool going forward.
UPDATE players SET singles_elo = GREATEST(singles_elo - 700, 100) WHERE gender = 'F';
