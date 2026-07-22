package bracket_test

import (
	"fmt"
	"testing"

	"table-tennis-backend/internal/domain/bracket"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
)

// passCountFixture builds an 8-player, 2-group, groups_elimination event whose
// group stage is fully played out. TeamA always wins, so each group's standings
// settle in participant order (p1 > p2 > p3 > p4), which keeps the "who
// advanced" assertions below deterministic.
func passCountFixture() ([]*player.Player, []event.Group, []event.Match) {
	players := make([]*player.Player, 8)
	elo := int16(1600)
	for i := range players {
		id := fmt.Sprintf("pc%d", i+1)
		players[i] = &player.Player{ID: id, FirstName: "P", LastName: id, Gender: "M", SinglesElo: elo}
		elo -= 50
	}

	groups := []event.Group{
		{ID: "gA", Name: "Open - Group A", Players: players[:4]},
		{ID: "gB", Name: "Open - Group B", Players: players[4:]},
	}

	var matches []event.Match
	mid := 0
	finishAllPairs := func(group []*player.Player) {
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				mid++
				matches = append(matches, event.Match{
					ID:         fmt.Sprintf("g%d", mid),
					Stage:      "group",
					Status:     "finished",
					WinnerTeam: "A",
					DivisionID: "d1",
					TeamA:      []*player.Player{group[i]},
					TeamB:      []*player.Player{group[j]},
				})
			}
		}
	}
	finishAllPairs(players[:4])
	finishAllPairs(players[4:])

	return players, groups, matches
}

func passCountDivisions() []*division.Division {
	return []*division.Division{
		{ID: "d1", Name: "Open", Category: "both", MinElo: 1, MaxElo: nil},
	}
}

// TestPassCount_InferredFromExistingMatches covers the regression fixed in
// fca20fe: once a knockout is under way, the bracket must be rebuilt at the
// size the existing matches imply, not at whatever GroupPassCount currently
// says. Here the stored config says 1 advances per group, but a quarterfinal
// round already exists containing 2 players from each group - so the rendered
// bracket has to stay at 4 advancing, or in-flight matches would be orphaned.
func TestPassCount_InferredFromExistingMatches(t *testing.T) {
	players, groups, matches := passCountFixture()

	// Quarterfinals already drawn with the top 2 of each group.
	matches = append(matches,
		event.Match{
			ID: "qf1", Stage: "quarterfinal", Status: "scheduled", DivisionID: "d1",
			TeamA: []*player.Player{players[0]}, TeamB: []*player.Player{players[5]},
		},
		event.Match{
			ID: "qf2", Stage: "quarterfinal", Status: "scheduled", DivisionID: "d1",
			TeamA: []*player.Player{players[1]}, TeamB: []*player.Player{players[4]},
		},
		// Neither of these may feed the tier-0 inference: one is another
		// division's, the other belongs to a losers bracket.
		event.Match{
			ID: "other", Stage: "quarterfinal", Status: "scheduled", DivisionID: "d2",
			TeamA: []*player.Player{players[2]}, TeamB: []*player.Player{players[6]},
		},
		event.Match{
			ID: "lb1", Stage: "loser_bracket", Status: "scheduled", DivisionID: "d1",
			TeamA: []*player.Player{players[3]}, TeamB: []*player.Player{players[7]},
		},
	)

	ev := &event.Event{
		ID:                    "t1",
		Type:                  "singles",
		Format:                "groups_elimination",
		EventCategory:         "open",
		Status:                "in_progress",
		KnockoutBracketsCount: 1,
		GroupPassCount:        1, // deliberately narrower than what's on the board
		Participants:          players,
		Groups:                groups,
		Matches:               matches,
	}

	br := bracket.BuildBracket(ev, passCountDivisions(), nil)
	dv := br.Divisions[0]
	if !dv.AllGroupsFinished {
		t.Fatalf("fixture should have every group finished, got AllGroupsFinished=false")
	}
	if len(dv.KnockoutBrackets) != 1 {
		t.Fatalf("expected 1 knockout bracket, got %d", len(dv.KnockoutBrackets))
	}
	if got := len(dv.KnockoutBrackets[0].Advancing); got != 4 {
		t.Errorf("expected the bracket to stay at 4 advancing (inferred from the existing quarterfinals), got %d", got)
	}
}

// TestPassCount_ConfigHonoredBeforeKnockoutStarts is the other half of the
// regression: the inference is gated on the event being under way. A draft
// event with the same data must still size the bracket from GroupPassCount.
func TestPassCount_ConfigHonoredBeforeKnockoutStarts(t *testing.T) {
	players, groups, matches := passCountFixture()

	ev := &event.Event{
		ID:                    "t2",
		Type:                  "singles",
		Format:                "groups_elimination",
		EventCategory:         "open",
		Status:                "draft",
		KnockoutBracketsCount: 1,
		GroupPassCount:        1,
		Participants:          players,
		Groups:                groups,
		Matches:               matches,
	}

	br := bracket.BuildBracket(ev, passCountDivisions(), nil)
	dv := br.Divisions[0]
	if got := len(dv.KnockoutBrackets[0].Advancing); got != 2 {
		t.Errorf("expected the configured 1-per-group (2 advancing) to be honored while drafting, got %d", got)
	}
}

// TestPassCount_NoKnockoutMatchesKeepsConfig covers the "nothing to infer from"
// branch: the event is in progress, but no knockout match exists yet, so the
// inference must return 0 and leave the configured pass count alone.
func TestPassCount_NoKnockoutMatchesKeepsConfig(t *testing.T) {
	players, groups, matches := passCountFixture()

	ev := &event.Event{
		ID:                    "t3",
		Type:                  "singles",
		Format:                "groups_elimination",
		EventCategory:         "open",
		Status:                "in_progress",
		KnockoutBracketsCount: 1,
		GroupPassCount:        2,
		Participants:          players,
		Groups:                groups,
		Matches:               matches,
	}

	br := bracket.BuildBracket(ev, passCountDivisions(), nil)
	dv := br.Divisions[0]
	if got := len(dv.KnockoutBrackets[0].Advancing); got != 4 {
		t.Errorf("expected the configured 2-per-group (4 advancing) to survive with no knockout matches to infer from, got %d", got)
	}
}

// TestPassCount_InferredForLosersTier exercises the tier>0 path, where the
// stage prefix ("tier1_") scopes the inference to the losers bracket. The
// configured losers pass count is 1, but a tier-1 semifinal already holds 2
// players from each group, so tier 1 must render at 4 advancing while the main
// bracket stays on its own (correct) configured size.
func TestPassCount_InferredForLosersTier(t *testing.T) {
	players, groups, matches := passCountFixture()

	matches = append(matches,
		// Main bracket, consistent with GroupPassCount=2.
		event.Match{
			ID: "qf1", Stage: "quarterfinal", Status: "scheduled", DivisionID: "d1",
			TeamA: []*player.Player{players[0]}, TeamB: []*player.Player{players[5]},
		},
		event.Match{
			ID: "qf2", Stage: "quarterfinal", Status: "scheduled", DivisionID: "d1",
			TeamA: []*player.Player{players[1]}, TeamB: []*player.Player{players[4]},
		},
		// Losers tier, drawn wider than LosersGroupPassCount claims.
		event.Match{
			ID: "t1sf1", Stage: "tier1_semifinal", Status: "scheduled", DivisionID: "d1",
			TeamA: []*player.Player{players[2]}, TeamB: []*player.Player{players[7]},
		},
		event.Match{
			ID: "t1sf2", Stage: "tier1_semifinal", Status: "scheduled", DivisionID: "d1",
			TeamA: []*player.Player{players[3]}, TeamB: []*player.Player{players[6]},
		},
	)

	ev := &event.Event{
		ID:                    "t4",
		Type:                  "singles",
		Format:                "groups_elimination",
		EventCategory:         "open",
		Status:                "finished",
		KnockoutBracketsCount: 2,
		GroupPassCount:        2,
		LosersGroupPassCount:  1, // narrower than the tier-1 matches on the board
		Participants:          players,
		Groups:                groups,
		Matches:               matches,
	}

	br := bracket.BuildBracket(ev, passCountDivisions(), nil)
	dv := br.Divisions[0]
	if len(dv.KnockoutBrackets) != 2 {
		t.Fatalf("expected 2 knockout brackets (main + tier 1), got %d", len(dv.KnockoutBrackets))
	}
	if got := len(dv.KnockoutBrackets[0].Advancing); got != 4 {
		t.Errorf("main bracket: expected 4 advancing, got %d", got)
	}
	if got := len(dv.KnockoutBrackets[1].Advancing); got != 4 {
		t.Errorf("tier 1 bracket: expected 4 advancing (inferred from the existing tier1_ matches), got %d", got)
	}
}
