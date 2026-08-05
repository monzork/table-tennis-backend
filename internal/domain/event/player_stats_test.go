package event

import (
	"table-tennis-backend/internal/domain/player"
	"testing"
)

func TestBuildPlayerEventStats(t *testing.T) {
	p1 := &player.Player{ID: "p1"}
	p2 := &player.Player{ID: "p2"}

	t.Run("counts wins, losses, sets and points across group and knockout stages", func(t *testing.T) {
		matches := []Match{
			{
				Status: "finished", Stage: "group", WinnerTeam: "A",
				TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2},
				Sets: []MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}, {Number: 2, ScoreA: 11, ScoreB: 7}},
			},
			{
				Status: "finished", Stage: "final", WinnerTeam: "B",
				TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2},
				Sets: []MatchSet{{Number: 1, ScoreA: 5, ScoreB: 11}, {Number: 2, ScoreA: 8, ScoreB: 11}},
			},
		}

		stats := BuildPlayerEventStats("p1", matches)
		if stats.Played != 2 || stats.Wins != 1 || stats.Losses != 1 {
			t.Errorf("expected 2 played, 1-1 record, got %+v", stats)
		}
		if stats.SetsWon != 2 || stats.SetsLost != 2 {
			t.Errorf("expected 2-2 sets, got %d-%d", stats.SetsWon, stats.SetsLost)
		}
		if stats.PointsWon != 35 || stats.PointsLost != 34 {
			t.Errorf("expected 35-34 points, got %d-%d", stats.PointsWon, stats.PointsLost)
		}
	})

	t.Run("ignores unfinished matches", func(t *testing.T) {
		matches := []Match{
			{Status: "in_progress", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}},
		}
		stats := BuildPlayerEventStats("p1", matches)
		if stats.Played != 0 {
			t.Errorf("expected 0 played for in-progress match, got %+v", stats)
		}
	})

	t.Run("returns zero value for a player with no matches", func(t *testing.T) {
		stats := BuildPlayerEventStats("ghost", []Match{})
		if stats.Played != 0 || stats.Wins != 0 {
			t.Errorf("expected zero stats, got %+v", stats)
		}
	})
}

func TestBuildAllPlayerEventStats(t *testing.T) {
	p1 := &player.Player{ID: "p1"}
	p2 := &player.Player{ID: "p2"}
	p3 := &player.Player{ID: "p3"}
	p4 := &player.Player{ID: "p4"}

	t.Run("credits every player on a doubles roster", func(t *testing.T) {
		matches := []Match{
			{
				Status: "finished", Stage: "group", WinnerTeam: "A",
				TeamA: []*player.Player{p1, p2}, TeamB: []*player.Player{p3, p4},
				Sets: []MatchSet{{Number: 1, ScoreA: 11, ScoreB: 9}},
			},
		}

		stats := BuildAllPlayerEventStats(matches)
		for _, id := range []string{"p1", "p2"} {
			if s := stats[id]; s.Wins != 1 || s.Losses != 0 {
				t.Errorf("expected %s to have 1 win, got %+v", id, s)
			}
		}
		for _, id := range []string{"p3", "p4"} {
			if s := stats[id]; s.Wins != 0 || s.Losses != 1 {
				t.Errorf("expected %s to have 1 loss, got %+v", id, s)
			}
		}
	})

	t.Run("matches single-player result for BuildPlayerEventStats", func(t *testing.T) {
		matches := []Match{
			{
				Status: "finished", Stage: "r16", WinnerTeam: "B",
				TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2},
				Sets: []MatchSet{{Number: 1, ScoreA: 9, ScoreB: 11}},
			},
		}
		all := BuildAllPlayerEventStats(matches)
		single := BuildPlayerEventStats("p1", matches)
		if all["p1"] != single {
			t.Errorf("expected batch and single-player results to match: %+v vs %+v", all["p1"], single)
		}
	})
}
