CREATE TABLE IF NOT EXISTS event_officials (
    event_id UUID NOT NULL,
    player_id UUID NOT NULL,
    pin TEXT NOT NULL,
    PRIMARY KEY (event_id, player_id)
);
