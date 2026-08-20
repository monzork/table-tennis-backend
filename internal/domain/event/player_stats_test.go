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

func TestBuildPlayerMatchDetails(t *testing.T) {
	p1 := &player.Player{ID: "p1", FirstName: "Ana"}
	p2 := &player.Player{ID: "p2", FirstName: "Beto"}

	t.Run("normalizes score and sets to the player's own perspective on both sides", func(t *testing.T) {
		matches := []Match{
			{
				Status: "finished", WinnerTeam: "A",
				TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2},
				Sets: []MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}, {Number: 2, ScoreA: 11, ScoreB: 7}},
			},
			{
				Status: "finished", WinnerTeam: "A",
				TeamA: []*player.Player{p2}, TeamB: []*player.Player{p1},
				Sets: []MatchSet{{Number: 1, ScoreA: 11, ScoreB: 6}},
			},
		}

		details := BuildPlayerMatchDetails("p1", matches)
		if len(details) != 2 {
			t.Fatalf("expected 2 matches, got %d", len(details))
		}

		won := details[0]
		if !won.Won || won.Opponent != "Beto " || won.SetsWon != 2 || won.SetsLost != 0 {
			t.Errorf("unexpected result as team A: %+v", won)
		}
		if won.Sets[0] != (PlayerSetScore{Number: 1, Own: 11, Opponent: 5}) {
			t.Errorf("unexpected set from team A perspective: %+v", won.Sets[0])
		}

		lost := details[1]
		if lost.Won || lost.Opponent != "Beto " || lost.SetsWon != 0 || lost.SetsLost != 1 {
			t.Errorf("unexpected result as team B: %+v", lost)
		}
		if lost.Sets[0] != (PlayerSetScore{Number: 1, Own: 6, Opponent: 11}) {
			t.Errorf("unexpected set from team B perspective: %+v", lost.Sets[0])
		}
	})

	t.Run("skips unfinished matches and matches the player wasn't in", func(t *testing.T) {
		matches := []Match{
			{Status: "in_progress", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}},
			{Status: "finished", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p2}, WinnerTeam: "A"},
		}
		if details := BuildPlayerMatchDetails("p1", matches); len(details) != 0 {
			t.Errorf("expected no matches, got %+v", details)
		}
	})
}

func TestBuildPlayerPendingMatchDetails(t *testing.T) {
	p1 := &player.Player{ID: "p1", FirstName: "Ana"}
	p2 := &player.Player{ID: "p2", FirstName: "Beto"}
	p3 := &player.Player{ID: "p3", FirstName: "Caro"}

	t.Run("includes scheduled/in_progress matches the player is in, excludes finished", func(t *testing.T) {
		table := 3
		matches := []Match{
			{ID: "m1", Status: "scheduled", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}},
			{ID: "m2", Status: "in_progress", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p1}, TableNumber: &table},
			{ID: "m3", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}},
			{ID: "m4", Status: "scheduled", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p3}},
		}

		details := BuildPlayerPendingMatchDetails("p1", "Men's Singles", matches)
		if len(details) != 2 {
			t.Fatalf("expected 2 pending matches, got %d: %+v", len(details), details)
		}
		if details[0].MatchID != "m1" || details[0].Opponent != "Beto " || details[0].Status != "scheduled" || details[0].EventName != "Men's Singles" {
			t.Errorf("unexpected first pending detail: %+v", details[0])
		}
		if details[1].MatchID != "m2" || details[1].Opponent != "Beto " || details[1].TableNumber == nil || *details[1].TableNumber != 3 {
			t.Errorf("unexpected second pending detail: %+v", details[1])
		}
	})

	t.Run("flags HasProposal and ProposedByMe correctly for both sides", func(t *testing.T) {
		proposerID := "p1"
		matches := []Match{
			{ID: "m1", Status: "in_progress", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}, ProposedByPlayerID: &proposerID},
		}

		mine := BuildPlayerPendingMatchDetails("p1", "Men's Singles", matches)
		if len(mine) != 1 || !mine[0].HasProposal || !mine[0].ProposedByMe {
			t.Errorf("expected proposer to see HasProposal+ProposedByMe, got %+v", mine)
		}

		theirs := BuildPlayerPendingMatchDetails("p2", "Men's Singles", matches)
		if len(theirs) != 1 || !theirs[0].HasProposal || theirs[0].ProposedByMe {
			t.Errorf("expected opponent to see HasProposal but not ProposedByMe, got %+v", theirs)
		}
	})

	t.Run("doubles: any team member counts as participant", func(t *testing.T) {
		matches := []Match{
			{ID: "m1", Status: "scheduled", TeamA: []*player.Player{p1, p3}, TeamB: []*player.Player{p2}},
		}
		details := BuildPlayerPendingMatchDetails("p3", "Men's Singles", matches)
		if len(details) != 1 {
			t.Fatalf("expected 1 pending match for doubles partner, got %d", len(details))
		}
	})

	t.Run("projects Elo win/loss from current ratings", func(t *testing.T) {
		favorite := &player.Player{ID: "fav", FirstName: "Fav", SinglesElo: 1800}
		underdog := &player.Player{ID: "dog", FirstName: "Dog", SinglesElo: 1200}
		matches := []Match{
			{ID: "m1", MatchType: "singles", Status: "scheduled", TeamA: []*player.Player{favorite}, TeamB: []*player.Player{underdog}},
		}

		details := BuildPlayerPendingMatchDetails("fav", "Men's Singles", matches)
		if len(details) != 1 {
			t.Fatalf("expected 1 pending match, got %d", len(details))
		}
		d := details[0]
		if d.ProjectedEloWin == nil || d.ProjectedEloLoss == nil {
			t.Fatalf("expected both projections set, got %+v", d)
		}
		if *d.ProjectedEloWin <= 0 {
			t.Errorf("expected a positive projected gain for the favorite winning, got %v", *d.ProjectedEloWin)
		}
		if *d.ProjectedEloLoss >= 0 {
			t.Errorf("expected a negative projected loss for the favorite losing, got %v", *d.ProjectedEloLoss)
		}
		// The favorite is expected to win, so an upset loss should cost more
		// than a routine win gains.
		if -*d.ProjectedEloLoss <= *d.ProjectedEloWin {
			t.Errorf("expected losing as favorite to cost more than winning gains, got win=%v loss=%v", *d.ProjectedEloWin, *d.ProjectedEloLoss)
		}
	})

	t.Run("nil projections when the opponent slot is still TBD", func(t *testing.T) {
		p1 := &player.Player{ID: "p1", SinglesElo: 1500}
		matches := []Match{
			{ID: "m1", MatchType: "singles", Status: "scheduled", TeamA: []*player.Player{p1}, TeamB: nil},
		}
		details := BuildPlayerPendingMatchDetails("p1", "Men's Singles", matches)
		if len(details) != 1 {
			t.Fatalf("expected 1 pending match, got %d", len(details))
		}
		if details[0].ProjectedEloWin != nil || details[0].ProjectedEloLoss != nil {
			t.Errorf("expected nil projections with no resolved opponent, got %+v", details[0])
		}
	})
}

func TestMatch_ProposedWinner(t *testing.T) {
	tests := []struct {
		label string
		sets  []MatchSet
		want  string
	}{
		{"A wins 2-0", []MatchSet{{ScoreA: 11, ScoreB: 5}, {ScoreA: 11, ScoreB: 8}}, "A"},
		{"B wins 2-1", []MatchSet{{ScoreA: 11, ScoreB: 5}, {ScoreA: 8, ScoreB: 11}, {ScoreA: 9, ScoreB: 11}}, "B"},
		{"no sets yet", nil, ""},
		{"set not yet decisive (no one reached 11)", []MatchSet{{ScoreA: 5, ScoreB: 3}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			m := Match{ProposedSets: tt.sets}
			if got := m.ProposedWinner(); got != tt.want {
				t.Errorf("ProposedWinner() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildPendingProposalPreviews(t *testing.T) {
	proposerID := "p1"
	favorite := &player.Player{ID: "p1", FirstName: "Fav", SinglesElo: 1800}
	underdog := &player.Player{ID: "p2", FirstName: "Dog", SinglesElo: 1200}

	t.Run("upset win by the underdog previews a positive delta for them", func(t *testing.T) {
		matches := []Match{
			{
				ID: "m1", MatchType: "singles", Status: "in_progress",
				TeamA: []*player.Player{favorite}, TeamB: []*player.Player{underdog},
				ProposedByPlayerID: &proposerID,
				ProposedSets:       []MatchSet{{ScoreA: 5, ScoreB: 11}, {ScoreA: 8, ScoreB: 11}},
			},
		}

		previews := BuildPendingProposalPreviews("p2", "Men's Singles", matches)
		if len(previews) != 1 {
			t.Fatalf("expected 1 preview, got %d: %+v", len(previews), previews)
		}
		p := previews[0]
		if !p.Won {
			t.Errorf("expected underdog to be marked as winning per the proposed sets, got %+v", p)
		}
		if p.CurrentElo != 1200 {
			t.Errorf("expected CurrentElo 1200, got %d", p.CurrentElo)
		}
		if p.EloDelta <= 0 {
			t.Errorf("expected a positive Elo delta for the winning underdog, got %v", p.EloDelta)
		}
		if p.Opponent != "Fav " {
			t.Errorf("expected opponent name 'Fav ', got %q", p.Opponent)
		}
	})

	t.Run("no preview when proposed sets don't yet decide a winner", func(t *testing.T) {
		matches := []Match{
			{
				ID: "m1", MatchType: "singles", Status: "in_progress",
				TeamA: []*player.Player{favorite}, TeamB: []*player.Player{underdog},
				ProposedByPlayerID: &proposerID,
				ProposedSets:       []MatchSet{{ScoreA: 5, ScoreB: 3}},
			},
		}
		if previews := BuildPendingProposalPreviews("p1", "Men's Singles", matches); len(previews) != 0 {
			t.Errorf("expected no previews for an indecisive proposal, got %+v", previews)
		}
	})

	t.Run("no preview when there's no active proposal", func(t *testing.T) {
		matches := []Match{
			{ID: "m1", MatchType: "singles", Status: "in_progress", TeamA: []*player.Player{favorite}, TeamB: []*player.Player{underdog}},
		}
		if previews := BuildPendingProposalPreviews("p1", "Men's Singles", matches); len(previews) != 0 {
			t.Errorf("expected no previews with no proposal, got %+v", previews)
		}
	})
}
