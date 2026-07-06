-- Add performance indexes on foreign keys
CREATE INDEX IF NOT EXISTS idx_tournaments_event_id ON tournaments(event_id);
CREATE INDEX IF NOT EXISTS idx_tournament_participants_tourn_id ON tournament_participants(tournament_id);
CREATE INDEX IF NOT EXISTS idx_tournament_participants_player_id ON tournament_participants(player_id);
CREATE INDEX IF NOT EXISTS idx_groups_tourn_id ON groups(tournament_id);
CREATE INDEX IF NOT EXISTS idx_group_participants_group_id ON group_participants(group_id);
CREATE INDEX IF NOT EXISTS idx_group_participants_player_id ON group_participants(player_id);
CREATE INDEX IF NOT EXISTS idx_matches_tourn_id ON matches(tournament_id);
CREATE INDEX IF NOT EXISTS idx_matches_team_a_1 ON matches(team_a_player_1_id);
CREATE INDEX IF NOT EXISTS idx_matches_team_b_1 ON matches(team_b_player_1_id);
CREATE INDEX IF NOT EXISTS idx_matches_division_id ON matches(division_id);
CREATE INDEX IF NOT EXISTS idx_matches_team_match_id ON matches(team_match_id);
CREATE INDEX IF NOT EXISTS idx_matches_referee_id ON matches(referee_id);
CREATE INDEX IF NOT EXISTS idx_match_sets_match_id ON match_sets(match_id);
