package migrations

import (
	"context"

	"github.com/monzork/table-tennis-backend/internal/db"
	"github.com/uptrace/bun"
)

func init() {
	db.Migrations.MustRegister(

		// ======================
		// UP
		// ======================
		func(ctx context.Context, db *bun.DB) error {

			// -------- Rules --------
			if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS rules (
				id           INTEGER PRIMARY KEY AUTOINCREMENT,
				name         TEXT NOT NULL,
				description  TEXT,
				created_at   TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at   TEXT,
				deleted_at   TEXT
			);
			`); err != nil {
				return err
			}

			// -------- Tournaments --------
			if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS tournaments (
				id           TEXT PRIMARY KEY,
				name         TEXT NOT NULL,
				description  TEXT,
				start_date   TEXT NOT NULL,
				end_date     TEXT NOT NULL,
				created_at   TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at   TEXT,
				deleted_at   TEXT
			);
			`); err != nil {
				return err
			}

			// -------- TournamentRules (M2M) --------
			if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS tournament_rules (
				tournament_id TEXT NOT NULL,
				rule_id       INTEGER NOT NULL,
				created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (tournament_id, rule_id),
				FOREIGN KEY (tournament_id) REFERENCES tournaments(id),
				FOREIGN KEY (rule_id) REFERENCES rules(id)
			);
			`); err != nil {
				return err
			}

			// -------- Matches --------
			if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS matches (
				id              TEXT PRIMARY KEY,
				tournament_id   TEXT NOT NULL,
				format          TEXT NOT NULL,
				status          TEXT NOT NULL DEFAULT 'scheduled',
				winner_team_id  TEXT,
				started_at      TEXT,
				finished_at     TEXT,
				created_at      TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at      TEXT,
				deleted_at      TEXT,
				FOREIGN KEY (tournament_id) REFERENCES tournaments(id)
			);
			`); err != nil {
				return err
			}

			// -------- MatchPlayers --------
			if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS match_players (
				match_id  TEXT NOT NULL,
				player_id TEXT NOT NULL,
				side      TEXT NOT NULL,
				PRIMARY KEY (match_id, player_id),
				FOREIGN KEY (match_id) REFERENCES matches(id)
			);
			`); err != nil {
				return err
			}

			// -------- MatchTeams --------
			if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS match_teams (
				id         TEXT PRIMARY KEY,
				match_id   TEXT NOT NULL,
				side       TEXT NOT NULL,
				created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (match_id) REFERENCES matches(id)
			);
			`); err != nil {
				return err
			}

			// -------- MatchTeamPlayers --------
			if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS match_team_players (
				match_team_id TEXT NOT NULL,
				player_id     TEXT NOT NULL,
				position      INTEGER NOT NULL,
				created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (match_team_id, player_id),
				FOREIGN KEY (match_team_id) REFERENCES match_teams(id)
			);
			`); err != nil {
				return err
			}

			// -------- MatchSets --------
			if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS match_sets (
				id         TEXT PRIMARY KEY,
				match_id   TEXT NOT NULL,
				number     INTEGER NOT NULL,
				score_a    INTEGER NOT NULL,
				score_b    INTEGER NOT NULL,
				created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TEXT,
				deleted_at TEXT,
				FOREIGN KEY (match_id) REFERENCES matches(id)
			);
			`); err != nil {
				return err
			}

			// -------- EloHistory --------
			if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS elo_history (
				player_id  TEXT NOT NULL,
				match_id   TEXT NOT NULL,
				before     INTEGER NOT NULL,
				after      INTEGER NOT NULL,
				created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			`); err != nil {
				return err
			}

			return nil
		},

		// ======================
		// DOWN
		// ======================
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.Exec(`
			DROP TABLE IF EXISTS elo_history;
			DROP TABLE IF EXISTS match_sets;
			DROP TABLE IF EXISTS match_team_players;
			DROP TABLE IF EXISTS match_teams;
			DROP TABLE IF EXISTS match_players;
			DROP TABLE IF EXISTS matches;
			DROP TABLE IF EXISTS tournament_rules;
			DROP TABLE IF EXISTS tournaments;
			DROP TABLE IF EXISTS rules;
			`)
			return err
		},
	)
}
