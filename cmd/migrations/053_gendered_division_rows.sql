-- Split each existing (gender-agnostic) division band into a men's and a
-- women's version. Existing rows become the men's bands; new rows are
-- inserted for women, shifted -700 to match the singles Elo normalization
-- applied in migration 051, so each gender's tiers reflect standing within
-- their own Elo distribution instead of a shared absolute scale.

UPDATE divisions SET name = 'Primera Division ♂', gender = 'M', display_order = 1 WHERE id = 'div-first';
UPDATE divisions SET name = 'Segunda Division ♂', gender = 'M', display_order = 3 WHERE id = 'div-second';
UPDATE divisions SET name = 'Tercera Division ♂', gender = 'M', display_order = 5 WHERE id = 'div-third';

INSERT INTO divisions (id, name, display_order, min_elo, max_elo, category, gender, color) VALUES
    ('div-first-f', 'Primera Division ♀', 2, 1300, NULL, 'both', 'F', '#C0C0C0'),
    ('div-second-f', 'Segunda Division ♀', 4, 600, 1299, 'both', 'F', '#CD7F32'),
    ('div-third-f', 'Tercera Division ♀', 6, 1, 599, 'both', 'F', '#4A90D9')
ON CONFLICT (id) DO NOTHING;
