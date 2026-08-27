-- Migration 041 (swap_event_and_tournament) renamed tournament_participants and
-- tournament_stage_rules to their event_* counterparts but missed this table,
-- leaving it as tournament_officials while all code references event_officials.
ALTER TABLE tournament_officials RENAME TO event_officials;
