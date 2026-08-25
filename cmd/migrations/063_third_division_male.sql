-- Splits 2nd Division (Men) into a 2nd/3rd tier, matching the original
-- gender-neutral div-first/div-second/div-third boundaries (2000/1300):
-- 1st: >=2000, 2nd: 1300-1999, 3rd: <1300. Women's bands are unchanged.
-- Only affects events with use_gender_divisions = true, added in migration
-- 062 -- every other event keeps using the untouched gender-agnostic bands.
UPDATE divisions SET min_elo = 1300, display_order = 11 WHERE id = 'div-second-male';

INSERT INTO divisions (id, name, display_order, min_elo, max_elo, category, gender, color) VALUES
    ('div-third-male', '3rd Division (Men)', 12, 0, 1300, 'both', 'M', '#4A90D9')
ON CONFLICT (id) DO NOTHING;

-- Keep the female bands consecutive after the men's three tiers in listings.
UPDATE divisions SET display_order = 13 WHERE id = 'div-first-female';
UPDATE divisions SET display_order = 14 WHERE id = 'div-second-female';
