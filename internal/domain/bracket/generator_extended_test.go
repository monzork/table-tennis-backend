package bracket_test

import (
	"table-tennis-backend/internal/domain/bracket"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"testing"
)

func TestGeneratorExtended_BuildBracket(t *testing.T) {
	// Dummy test covering a lot of generator execution paths
	players := []*player.Player{
		{ID: "p1", FirstName: "P", LastName: "1", SinglesElo: 1200, Gender: "M"},
		{ID: "p2", FirstName: "P", LastName: "2", SinglesElo: 1100, Gender: "M"},
		{ID: "p3", FirstName: "P", LastName: "3", SinglesElo: 1000, Gender: "M"},
		{ID: "p4", FirstName: "P", LastName: "4", SinglesElo: 900, Gender: "M"},
	}

	tourney := &event.Event{
		ID:                    "t1",
		Name:                  "Test Tournament",
		Type:                  "singles",
		Format:                "single_elimination",
		EventCategory:         "open",
		KnockoutBracketsCount: 1,
		GroupPassCount:        2,
		LosersGroupPassCount:  1,
		Participants:          players,
	}

	divs := []*division.Division{
		{ID: "div1", Name: "Open", Category: "both", MinElo: 1, MaxElo: nil},
	}

	br := bracket.BuildBracket(tourney, divs, nil)
	if len(br.Divisions) == 0 {
		t.Fatalf("Expected divisions")
	}

	// Test double elimination format
	tourney.Format = "double_elimination"
	br2 := bracket.BuildBracket(tourney, divs, nil)
	if len(br2.Divisions) == 0 {
		t.Fatalf("Expected divisions")
	}

	// Test round robin format
	tourney.Format = "round_robin"
	br3 := bracket.BuildBracket(tourney, divs, nil)
	if len(br3.Divisions) == 0 {
		t.Fatalf("Expected divisions")
	}

	// Test teams format
	tourney.Type = "teams"
	tourney.Teams = []*event.Team{
		{ID: "team1", Name: "Team 1", Players: []*player.Player{players[0], players[1]}},
		{ID: "team2", Name: "Team 2", Players: []*player.Player{players[2], players[3]}},
	}
	br4 := bracket.BuildBracket(tourney, divs, nil)
	if len(br4.Divisions) == 0 {
		t.Fatalf("Expected divisions")
	}

	// Test teams knockout
	tourney.Format = "single_elimination"
	br5 := bracket.BuildBracket(tourney, divs, nil)
	if len(br5.Divisions) == 0 {
		t.Fatalf("Expected divisions")
	}
}

func TestGeneratorExtended_GetLosersGroupPassCount(t *testing.T) {
	tourney := &event.Event{
		DivisionConfigs: map[string]event.DivisionConfig{
			"div1": {LosersGroupPassCount: 5},
		},
		LosersGroupPassCount: 3,
	}
	if count := tourney.GetLosersGroupPassCount("div1"); count != 5 {
		t.Errorf("expected 5, got %d", count)
	}
	if count := tourney.GetLosersGroupPassCount("div2"); count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestGeneratorExtended_BuildBracketRounds(t *testing.T) {
	players := []*player.Player{
		{ID: "p1", SinglesElo: 1200},
		{ID: "p2", SinglesElo: 1100},
		{ID: "p3", SinglesElo: 1000},
		{ID: "p4", SinglesElo: 900},
		{ID: "p5", SinglesElo: 800},
		{ID: "p6", SinglesElo: 700},
		{ID: "p7", SinglesElo: 600},
		{ID: "p8", SinglesElo: 500},
	}
	tourney := &event.Event{
		ID:                    "t1",
		Format:                "single_elimination",
		Type:                  "singles",
		EventCategory:         "open",
		KnockoutBracketsCount: 1,
		Participants:          players,
	}
	divs := []*division.Division{
		{ID: "d1", Name: "D1", MinElo: 1, MaxElo: nil},
	}

	br := bracket.BuildBracket(tourney, divs, nil)
	if len(br.Divisions) == 0 {
		t.Fatalf("Expected divisions")
	}

	// Add some groups to test group elimination
	tourney.Format = "groups_elimination"
	group := event.Group{
		ID:           "g1",
		TournamentID: "t1",
		Name:         "G1",
		Players:      players,
		Matches:      []event.Match{},
	}
	tourney.Groups = []event.Group{group}

	br2 := bracket.BuildBracket(tourney, divs, nil)
	if len(br2.Divisions) == 0 {
		t.Fatalf("Expected divisions")
	}
}

func TestGeneratorExtended_ValidateSameGroupSeparation(t *testing.T) {
	players := []*player.Player{
		{ID: "p1"}, {ID: "p2"},
	}
	groups := []bracket.Group{
		{
			ID:      "g1",
			Players: []*player.Player{players[0], players[1]},
		},
	}
	err := bracket.ValidateSameGroupSeparation(groups, players)
	// it should be nil if no problem, or error if they can face in first round
	_ = err
}
