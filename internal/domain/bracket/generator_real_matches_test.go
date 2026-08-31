package bracket_test

import (
	"table-tennis-backend/internal/domain/bracket"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"testing"
)

// TestBuildBracketRounds_RealMatchesOverrideMismatchedSeeding is a
// regression test for a production bug (division "Tercera Division"): a
// knockout bracket that was fully played and had a real champion rendered
// as BYE for every round past the first, because the players list handed
// in (recomputed off *current* group standings/GroupPassCount) didn't
// reproduce the pairing actually drawn/played. See the identical fix and
// comment on firstRoundPairsFromRealMatches.
func TestBuildBracketRounds_RealMatchesOverrideMismatchedSeeding(t *testing.T) {
	p := make(map[string]*player.Player, 8)
	for i := 1; i <= 8; i++ {
		id := "p" + string(rune('0'+i))
		p[id] = &player.Player{ID: id, FirstName: id}
	}
	// players is sorted p1..p8, but the real first-round pairing below is
	// p1-p2, p3-p4, p5-p6, p7-p8 -- not the seeded arrangement that order
	// would imply, and not reconstructable by re-deriving seeds.
	players := []*player.Player{p["p1"], p["p2"], p["p3"], p["p4"], p["p5"], p["p6"], p["p7"], p["p8"]}

	win := []event.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}}
	ev := &event.Event{
		Matches: []event.Match{
			{ID: "qf-1", Stage: "quarterfinal", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p2"]}, Sets: win},
			{ID: "qf-2", Stage: "quarterfinal", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p3"]}, TeamB: []*player.Player{p["p4"]}, Sets: win},
			{ID: "qf-3", Stage: "quarterfinal", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p5"]}, TeamB: []*player.Player{p["p6"]}, Sets: win},
			{ID: "qf-4", Stage: "quarterfinal", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p7"]}, TeamB: []*player.Player{p["p8"]}, Sets: win},
			{ID: "sf-1", Stage: "semifinal", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p3"]}, Sets: win},
			{ID: "sf-2", Stage: "semifinal", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p5"]}, TeamB: []*player.Player{p["p7"]}, Sets: win},
			{ID: "final-1", Stage: "final", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p5"]}, Sets: win},
		},
	}

	rounds := bracket.BuildBracketRoundsForTest(ev, "d3", players, 0)

	if len(rounds) != 4 {
		names := make([]string, len(rounds))
		for i, r := range rounds {
			names[i] = r.Name
		}
		t.Fatalf("expected 4 rounds (QF, SF, Final, Champion), got %d: %v", len(rounds), names)
	}
	champRound := rounds[len(rounds)-1]
	if champRound.Name != "Champion" {
		t.Fatalf("expected last round Champion, got %s", champRound.Name)
	}
	if champRound.Matches[0].Player1 == nil || champRound.Matches[0].Player1.Player == nil {
		t.Fatalf("expected a resolved champion, got BYE/unresolved: %+v", champRound.Matches[0])
	}
	if got := champRound.Matches[0].Player1.Player.ID; got != "p1" {
		t.Errorf("expected champion p1 (winner of every real match on its path), got %s", got)
	}
}
