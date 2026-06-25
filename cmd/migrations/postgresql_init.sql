-- Combined PostgreSQL Initialization Script
-- Consolidated from migrations 001 through 008
-- Enable UUID extension if needed
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- 1. Table: players
CREATE TABLE IF NOT EXISTS players (
    id UUID PRIMARY KEY,
    first_name TEXT NOT NULL,
    second_name TEXT NOT NULL DEFAULT '',
    last_name TEXT NOT NULL,
    second_last_name TEXT NOT NULL DEFAULT '',
    birthdate TIMESTAMP NOT NULL,
    gender TEXT NOT NULL DEFAULT 'M',
    singles_elo INTEGER NOT NULL DEFAULT 1000,
    doubles_elo INTEGER NOT NULL DEFAULT 1000,
    country TEXT NOT NULL,
    department TEXT NOT NULL DEFAULT '',
    whatsapp_number TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);
-- 2. Table: tournaments
CREATE TABLE IF NOT EXISTS tournaments (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'singles',
    format TEXT NOT NULL DEFAULT 'elimination',
    status TEXT NOT NULL DEFAULT 'in_progress',
    event_category TEXT NOT NULL DEFAULT 'open',
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    group_pass_count INTEGER NOT NULL DEFAULT 2,
    registration_open BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);
-- 3. Table: rules (general rules)
CREATE TABLE IF NOT EXISTS rules (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);
-- 4. Table: groups
CREATE TABLE IF NOT EXISTS groups (
    id UUID PRIMARY KEY,
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    name TEXT NOT NULL
);
-- 5. Table: group_participants
CREATE TABLE IF NOT EXISTS group_participants (
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, player_id)
);
-- 6. Table: matches
CREATE TABLE IF NOT EXISTS matches (
    id UUID PRIMARY KEY,
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    match_type TEXT NOT NULL DEFAULT 'singles',
    -- 'singles' or 'doubles'
    team_a_player_1_id UUID NOT NULL REFERENCES players(id),
    team_a_player_2_id UUID REFERENCES players(id),
    team_b_player_1_id UUID NOT NULL REFERENCES players(id),
    team_b_player_2_id UUID REFERENCES players(id),
    status TEXT NOT NULL DEFAULT 'scheduled',
    winner_team TEXT,
    -- 'A', 'B', or NULL
    stage TEXT NOT NULL DEFAULT 'final',
    group_id UUID REFERENCES groups(id),
    next_match_id UUID REFERENCES matches(id),
    next_match_slot TEXT DEFAULT 'A',
    -- 'A' or 'B'
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);
-- 7. Table: match_sets
CREATE TABLE IF NOT EXISTS match_sets (
    id UUID PRIMARY KEY,
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    set_number INTEGER NOT NULL,
    score_a INTEGER NOT NULL DEFAULT 0,
    score_b INTEGER NOT NULL DEFAULT 0,
    UNIQUE(match_id, set_number)
);
-- 8. Table: tournament_participants (Registration record)
CREATE TABLE IF NOT EXISTS tournament_participants (
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    elo_before_singles INTEGER,
    elo_before_doubles INTEGER,
    elo_after_singles INTEGER,
    elo_after_doubles INTEGER,
    PRIMARY KEY (tournament_id, player_id)
);
-- 9. Table: tournament_stage_rules
CREATE TABLE IF NOT EXISTS tournament_stage_rules (
    id UUID PRIMARY KEY,
    tournament_id UUID NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    stage TEXT NOT NULL,
    best_of INTEGER NOT NULL DEFAULT 5,
    points_to_win INTEGER NOT NULL DEFAULT 11,
    points_margin INTEGER NOT NULL DEFAULT 2
);
-- 10. Table: admins
CREATE TABLE IF NOT EXISTS admins (
    id UUID PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);
-- 11. Table: divisions
CREATE TABLE IF NOT EXISTS divisions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    min_elo INTEGER NOT NULL,
    max_elo INTEGER,
    category TEXT NOT NULL DEFAULT 'both', -- 'singles', 'doubles', 'both'
    color TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);

-- Seed default divisions
INSERT INTO divisions (id, name, display_order, min_elo, max_elo, category, color) VALUES
    ('div-champion', 'Champion', 1, 2000, NULL, 'both', '#FFD700'),
    ('div-first', 'First Division', 2, 1600, 1999, 'both', '#C0C0C0'),
    ('div-second', 'Second Division', 3, 1300, 1599, 'both', '#CD7F32'),
    ('div-third', 'Third Division', 4, 1000, 1299, 'both', '#4A90D9'),
    ('div-fourth', 'Fourth Division', 5, 0, 999, 'both', '#7B8794'),
    ('none', 'No Division', 99, 0, 9999, 'both', '#7B8794')
ON CONFLICT (id) DO NOTHING;

-- 12. Table: events
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    division_id TEXT NOT NULL REFERENCES divisions(id),
    skip_elo BOOLEAN NOT NULL DEFAULT FALSE,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);

-- Alter table tournaments to add event_id and skip_elo
ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS event_id UUID REFERENCES events(id) ON DELETE SET NULL;
ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS skip_elo BOOLEAN NOT NULL DEFAULT FALSE;