-- Adds a per-event opt-in flag so bracket generation can use gender-specific
-- division bands (see internal/domain/bracket/generator.go) instead of the
-- shared gender-agnostic bands, without affecting any existing event -- the
-- flag defaults to false, which is today's exact behavior.
ALTER TABLE events ADD COLUMN IF NOT EXISTS use_gender_divisions boolean NOT NULL DEFAULT false;

-- New gender-specific division bands. These are additive -- none of the
-- existing gender-agnostic rows (div-champion/div-first/div-second/div-third/
-- div-fourth/none) are modified, so any event/tournament that doesn't opt in
-- via use_gender_divisions above keeps seeing exactly the bands it does today.
INSERT INTO divisions (id, name, display_order, min_elo, max_elo, category, gender, color) VALUES
    ('div-first-male',    '1st Division (Men)',   10, 2000, NULL, 'both', 'M', '#C0C0C0'),
    ('div-second-male',   '2nd Division (Men)',   11, 0,    2000, 'both', 'M', '#7B8794'),
    ('div-first-female',  '1st Division (Women)', 12, 1300, NULL, 'both', 'F', '#C0C0C0'),
    ('div-second-female', '2nd Division (Women)', 13, 0,    1300, 'both', 'F', '#7B8794')
ON CONFLICT (id) DO NOTHING;
