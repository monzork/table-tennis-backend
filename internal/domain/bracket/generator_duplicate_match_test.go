package bracket_test

import (
	"table-tennis-backend/internal/domain/bracket"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"testing"
)

// TestBuildBracketRounds_DuplicateFirstRoundMatchDoesNotDuplicatePlayer is a
// regression test for a production bug (2nd Division, "II Ranking
// Nacional"): a stale/duplicate quarterfinal match row for the same player
// (bad DivisionID tagging leaving behind an extra record) caused that player
// to be rendered as the winner of two different sibling bracket matches in
// the same round. See firstRoundPairsFromRealMatches's provider-dedup guard.
func TestBuildBracketRounds_DuplicateFirstRoundMatchDoesNotDuplicatePlayer(t *testing.T) {
	p := make(map[string]*player.Player, 4)
	for i := 1; i <= 4; i++ {
		id := "p" + string(rune('0'+i))
		p[id] = &player.Player{ID: id, FirstName: id}
	}
	players := []*player.Player{p["p1"], p["p2"], p["p3"], p["p4"]}

	win := []event.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}}
	ev := &event.Event{
		Matches: []event.Match{
			{ID: "qf-1", Stage: "quarterfinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p2"]}, Sets: win},
			// Stale duplicate of qf-1 with a different opponent: p1 already
			// has a provider slot from qf-1 above, so this row must be
			// skipped rather than giving p1 a second slot.
			{ID: "qf-1-dup", Stage: "quarterfinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p3"]}, Sets: win},
			{ID: "sf-1", Stage: "semifinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p4"]}, Sets: win},
		},
	}

	rounds := bracket.BuildBracketRoundsForTest(ev, "d1", players, 0)

	seen := map[string]int{}
	for _, r := range rounds {
		for _, m := range r.Matches {
			for _, slot := range []*bracket.MatchSlot{m.Player1, m.Player2} {
				if slot != nil && slot.Player != nil {
					seen[slot.Player.ID]++
				}
			}
		}
	}
	for id, count := range seen {
		if count > 3 {
			t.Errorf("player %s appears in %d bracket slots across %d rounds; a single-elimination bracket must not place the same player into multiple sibling matches from a duplicate match row", id, count, len(rounds))
		}
	}
}
