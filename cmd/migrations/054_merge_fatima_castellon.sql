-- Merge duplicate player "Fatima Castellon" records, keeping the one with
-- singles_elo 1037 (Fatima Guadalupe Castellón Navarro,
-- 49c91ecb-f807-4984-b410-3dfd9a30885e) and folding the other
-- (1e31eecd-6bef-4d63-8b28-839879f34ff2) into it. The two records have no
-- overlapping events/teams/groups, so reassigning references is conflict-free.

UPDATE matches SET team_a_player_1_id = '49c91ecb-f807-4984-b410-3dfd9a30885e' WHERE team_a_player_1_id = '1e31eecd-6bef-4d63-8b28-839879f34ff2';
UPDATE matches SET team_a_player_2_id = '49c91ecb-f807-4984-b410-3dfd9a30885e' WHERE team_a_player_2_id = '1e31eecd-6bef-4d63-8b28-839879f34ff2';
UPDATE matches SET team_b_player_1_id = '49c91ecb-f807-4984-b410-3dfd9a30885e' WHERE team_b_player_1_id = '1e31eecd-6bef-4d63-8b28-839879f34ff2';
UPDATE matches SET team_b_player_2_id = '49c91ecb-f807-4984-b410-3dfd9a30885e' WHERE team_b_player_2_id = '1e31eecd-6bef-4d63-8b28-839879f34ff2';
UPDATE matches SET proposed_by_player_id = '49c91ecb-f807-4984-b410-3dfd9a30885e' WHERE proposed_by_player_id = '1e31eecd-6bef-4d63-8b28-839879f34ff2';

UPDATE event_participants SET player_id = '49c91ecb-f807-4984-b410-3dfd9a30885e' WHERE player_id = '1e31eecd-6bef-4d63-8b28-839879f34ff2';
UPDATE team_players SET player_id = '49c91ecb-f807-4984-b410-3dfd9a30885e' WHERE player_id = '1e31eecd-6bef-4d63-8b28-839879f34ff2';
UPDATE group_participants SET player_id = '49c91ecb-f807-4984-b410-3dfd9a30885e' WHERE player_id = '1e31eecd-6bef-4d63-8b28-839879f34ff2';

DELETE FROM players WHERE id = '1e31eecd-6bef-4d63-8b28-839879f34ff2';
