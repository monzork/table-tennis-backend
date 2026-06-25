CREATE TABLE IF NOT EXISTS tournament_officials (
    tournament_id UUID NOT NULL,
    player_id UUID NOT NULL,
    pin TEXT NOT NULL,
    PRIMARY KEY (tournament_id, player_id)
);
