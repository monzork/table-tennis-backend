package bracket_test

import (
	"table-tennis-backend/internal/domain/bracket"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"testing"
)

func TestBracketGenerator_LosersGroupPassCount(t *testing.T) {
	// Setup players
	players := []*player.Player{
		{ID: "p1", FirstName: "Player", LastName: "1", SinglesElo: 1200, Gender: "M"},
		{ID: "p2", FirstName: "Player", LastName: "2", SinglesElo: 1100, Gender: "M"},
		{ID: "p3", FirstName: "Player", LastName: "3", SinglesElo: 1000, Gender: "M"},
		{ID: "p4", FirstName: "Player", LastName: "4", SinglesElo: 900, Gender: "M"},
		{ID: "p5", FirstName: "Player", LastName: "5", SinglesElo: 800, Gender: "M"},
		{ID: "p6", FirstName: "Player", LastName: "6", SinglesElo: 700, Gender: "M"},
		{ID: "p7", FirstName: "Player", LastName: "7", SinglesElo: 600, Gender: "M"},
		{ID: "p8", FirstName: "Player", LastName: "8", SinglesElo: 500, Gender: "M"},
	}

	tourney := &event.Event{
		ID:                    "t1",
		Name:                  "Test Tournament",
		Type:                  "singles",
		Format:                "groups_elimination",
		EventCategory:         "open",
		KnockoutBracketsCount: 2,
		GroupPassCount:        2,
		LosersGroupPassCount:  1,
		Participants:          players,
	}

	group1 := event.Group{
		ID:           "g1",
		TournamentID: "t1",
		Name:         "Open - Group A",
		Players:      []*player.Player{players[0], players[1], players[2], players[3]},
		Matches:      []event.Match{},
	}

	group1.Matches = append(group1.Matches, createFinishedMatch("m1", players[0], players[1], "A"))
	group1.Matches = append(group1.Matches, createFinishedMatch("m2", players[0], players[2], "A"))
	group1.Matches = append(group1.Matches, createFinishedMatch("m3", players[0], players[3], "A"))
	group1.Matches = append(group1.Matches, createFinishedMatch("m4", players[1], players[2], "A"))
	group1.Matches = append(group1.Matches, createFinishedMatch("m5", players[1], players[3], "A"))
	group1.Matches = append(group1.Matches, createFinishedMatch("m6", players[2], players[3], "A"))

	group2 := event.Group{
		ID:           "g2",
		TournamentID: "t1",
		Name:         "Open - Group B",
		Players:      []*player.Player{players[4], players[5], players[6], players[7]},
		Matches:      []event.Match{},
	}
	group2.Matches = append(group2.Matches, createFinishedMatch("m7", players[4], players[5], "A"))
	group2.Matches = append(group2.Matches, createFinishedMatch("m8", players[4], players[6], "A"))
	group2.Matches = append(group2.Matches, createFinishedMatch("m9", players[4], players[7], "A"))
	group2.Matches = append(group2.Matches, createFinishedMatch("m10", players[5], players[6], "A"))
	group2.Matches = append(group2.Matches, createFinishedMatch("m11", players[5], players[7], "A"))
	group2.Matches = append(group2.Matches, createFinishedMatch("m12", players[6], players[7], "A"))

	tourney.Groups = []event.Group{group1, group2}
	tourney.Matches = append(tourney.Matches, group1.Matches...)
	tourney.Matches = append(tourney.Matches, group2.Matches...)

	divs := []*division.Division{
		{ID: "div1", Name: "Open", Category: "both", MinElo: 1, MaxElo: nil},
	}

	br := bracket.BuildBracket(tourney, divs, map[string]string{})
	views := br.Divisions

	if len(views) != 1 {
		t.Fatalf("Expected 1 division view, got %d", len(views))
	}

	divView := views[0]
	if len(divView.KnockoutBrackets) != 2 {
		t.Fatalf("Expected 2 brackets (tiers), got %d", len(divView.KnockoutBrackets))
	}

	tier1 := divView.KnockoutBrackets[0]
	tier2 := divView.KnockoutBrackets[1]

	if len(tier1.Rounds[0].Matches) != 2 {
		t.Errorf("Tier 1 should have 2 matches in round 1, got %d", len(tier1.Rounds[0].Matches))
	}

	if len(tier2.Rounds[0].Matches) != 1 {
		t.Errorf("Tier 2 should have 1 match in round 1, got %d", len(tier2.Rounds[0].Matches))
	}

	tourney.DivisionConfigs = map[string]event.DivisionConfig{
		"div1": {LosersGroupPassCount: 2},
	}

	br = bracket.BuildBracket(tourney, divs, map[string]string{})
	views = br.Divisions
	divView = views[0]
	tier2 = divView.KnockoutBrackets[1]

	if len(tier2.Rounds[0].Matches) != 2 {
		t.Errorf("With override, Tier 2 should have 2 matches in round 1, got %d (Rounds len: %d, tierAdvancing len: %d)", len(tier2.Rounds[0].Matches), len(tier2.Rounds), len(tier2.Advancing))
		t.Logf("Event DivisionConfigs: %v", tourney.DivisionConfigs)
		t.Logf("GetLosersGroupPassCount('div1'): %d", tourney.GetLosersGroupPassCount("div1"))
		t.Logf("Actual divID in view: %s", divView.ID)
		t.Logf("Actual tier2 pass count used: %d", tourney.GetLosersGroupPassCount(divView.ID))
	}
}

func createFinishedMatch(id string, p1, p2 *player.Player, winner string) event.Match {
	scoreA := 0
	scoreB := 0
	if winner == "A" {
		scoreA = 11
		scoreB = 5
	} else {
		scoreA = 5
		scoreB = 11
	}
	return event.Match{
		ID:         id,
		TeamA:      []*player.Player{p1},
		TeamB:      []*player.Player{p2},
		Status:     "finished",
		Stage:      "group",
		WinnerTeam: winner,
		Sets: []event.MatchSet{
			{ScoreA: scoreA, ScoreB: scoreB},
			{ScoreA: scoreA, ScoreB: scoreB},
			{ScoreA: scoreA, ScoreB: scoreB},
		},
	}
}
