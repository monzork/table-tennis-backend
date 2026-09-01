package bracket_test

import (
	"table-tennis-backend/internal/domain/bracket"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"testing"
)

// TestBuildBracketRounds_PreviewSemifinalRespectsQuarterAdjacency is a
// regression test for a production bug (2nd/3rd Division, "II Ranking
// Nacional"): once all four quarterfinals are real/finished but no
// semifinal match exists yet, the semifinal *preview* must pair the two
// quarters that share the same half of the draw -- not whatever order the
// QF match rows happen to come back from the DB in.
//
// For 8 seeded players, getSeedingArrangement(8) is [1,8,5,4,3,6,7,2], so
// the canonical quarterfinals are (seed1 vs seed8) and (seed5 vs seed4) in
// one half, (seed3 vs seed6) and (seed7 vs seed2) in the other. The correct
// semifinals are therefore (QF1 winner vs QF2 winner) and (QF3 winner vs
// QF4 winner). The QF match rows are deliberately inserted out of that
// order (QF3 first) to reproduce the bug: without tracking true
// bracket-tree position (MatchSlot.BracketPos), the naive "pair consecutive
// winners" fallback cross-wired quarters from opposite halves instead.
func TestBuildBracketRounds_PreviewSemifinalRespectsQuarterAdjacency(t *testing.T) {
	p := make(map[string]*player.Player, 8)
	for i := 1; i <= 8; i++ {
		id := "p" + string(rune('0'+i))
		p[id] = &player.Player{ID: id, FirstName: id}
	}
	// players[i] has seed i+1, matching seedOf's convention in
	// firstRoundPairsFromRealMatches.
	players := []*player.Player{p["p1"], p["p2"], p["p3"], p["p4"], p["p5"], p["p6"], p["p7"], p["p8"]}

	win := []event.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}}
	ev := &event.Event{
		Matches: []event.Match{
			// Scrambled DB row order: QF3 (seed3 vs seed6) before QF2
			// (seed5 vs seed4) before QF1 (seed1 vs seed8) before QF4
			// (seed7 vs seed2).
			{ID: "qf3", Stage: "quarterfinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p3"]}, TeamB: []*player.Player{p["p6"]}, Sets: win},
			{ID: "qf2", Stage: "quarterfinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p5"]}, TeamB: []*player.Player{p["p4"]}, Sets: win},
			{ID: "qf1", Stage: "quarterfinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p8"]}, Sets: win},
			{ID: "qf4", Stage: "quarterfinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p7"]}, TeamB: []*player.Player{p["p2"]}, Sets: win},
			// No semifinal match yet -- this is the preview the admin would
			// see and use to create the real SF match from.
		},
	}

	rounds := bracket.BuildBracketRoundsForTest(ev, "d1", players, 0)

	var sfRound *bracket.Round
	for i := range rounds {
		if rounds[i].Name == "Semi-Finals" {
			sfRound = &rounds[i]
			break
		}
	}
	if sfRound == nil {
		t.Fatalf("expected a Semi-Finals round, got: %+v", rounds)
	}

	seen := map[string]bool{}
	for _, m := range sfRound.Matches {
		a, b := "", ""
		if m.Player1 != nil && m.Player1.Player != nil {
			a = m.Player1.Player.ID
		}
		if m.Player2 != nil && m.Player2.Player != nil {
			b = m.Player2.Player.ID
		}
		seen[a+"-"+b] = true
		seen[b+"-"+a] = true
	}

	// QF1 (seed1) and QF2 (seed5) share a half -> should meet in the semis.
	if !seen["p1-p5"] {
		t.Errorf("expected QF1 winner (p1) to face QF2 winner (p5) in the semifinal preview, semis were: %+v", sfRound.Matches)
	}
	// QF3 (seed3) and QF4 (seed7) share the other half.
	if !seen["p3-p7"] {
		t.Errorf("expected QF3 winner (p3) to face QF4 winner (p7) in the semifinal preview, semis were: %+v", sfRound.Matches)
	}
}
