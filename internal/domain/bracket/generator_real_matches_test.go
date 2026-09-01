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

// TestGroupSlotsByRealMatches_CrossedPairingSortedBackToPosition is a
// regression test for a production bug affecting the admin and public
// bracket/draw views: when the real next-round matches pair non-adjacent
// slots (e.g. slot 0's winner actually played slot 3's, not slot 1's), the
// pairing function used to return pairs in whatever arbitrary order the
// underlying matches happened to be recorded in the DB -- so a player's own
// next match could render at the top of the column right after their
// previous match was drawn at the bottom, with no visual relationship
// between the two. The pairing itself (who plays whom) was always correct;
// only the rendering order needed to go back to bracket position.
func TestGroupSlotsByRealMatches_CrossedPairingSortedBackToPosition(t *testing.T) {
	mk := func(id string) *bracket.MatchSlot {
		return &bracket.MatchSlot{Player: &player.Player{ID: id, FirstName: id}}
	}
	slots := []*bracket.MatchSlot{mk("s0"), mk("s1"), mk("s2"), mk("s3")}

	// Real matches arrive in an order that crosses the slots (s0 played s3,
	// found before s1 played s2) and would, without sorting, put the s1/s2
	// pair first in the output.
	nextMatches := []*event.Match{
		{TeamA: []*player.Player{slots[1].Player}, TeamB: []*player.Player{slots[2].Player}},
		{TeamA: []*player.Player{slots[0].Player}, TeamB: []*player.Player{slots[3].Player}},
	}

	pairs := bracket.GroupSlotsByRealMatchesForTest(slots, nextMatches)

	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d: %+v", len(pairs), pairs)
	}
	// The pair containing s0 (the lowest original slot index) must be
	// sorted first, even though its real match was recorded second.
	first := pairs[0]
	gotIDs := map[string]bool{}
	if first.P1 != nil && first.P1.Player != nil {
		gotIDs[first.P1.Player.ID] = true
	}
	if first.P2 != nil && first.P2.Player != nil {
		gotIDs[first.P2.Player.ID] = true
	}
	if !gotIDs["s0"] || !gotIDs["s3"] {
		t.Errorf("expected the first pair to be s0/s3 (lowest original index), got %+v", pairs)
	}
}
