-- Tracks how many consecutive federation-endorsed tournaments a player has
-- concluded without enrolling in any of their events, so an inactivity Elo
-- decay can be applied and the player flagged inactive on the public
-- ranking. See internal/application/tournament.ApplyInactivityDecayUseCase.
ALTER TABLE players ADD COLUMN IF NOT EXISTS missed_federated_tournaments SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE players ADD COLUMN IF NOT EXISTS inactive BOOLEAN NOT NULL DEFAULT FALSE;

-- Single-row table of admin-configurable inactivity-decay parameters:
-- tournament_threshold tournaments missed before a penalty applies (and the
-- player is flagged inactive), elo_penalty points lost per threshold
-- crossed, and elo_floor below which decay stops.
CREATE TABLE IF NOT EXISTS inactivity_settings (
    id                    TEXT PRIMARY KEY DEFAULT 'default',
    tournament_threshold  SMALLINT NOT NULL DEFAULT 4,
    elo_penalty           SMALLINT NOT NULL DEFAULT 50,
    elo_floor             SMALLINT NOT NULL DEFAULT 1900
);
INSERT INTO inactivity_settings (id) VALUES ('default') ON CONFLICT (id) DO NOTHING;
