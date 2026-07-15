package handler

import (
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/bracket"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"testing"
)

func TestBuildBracket_GroupPassCount(t *testing.T) {
	tmap := make(map[string]string)

	div1 := &division.Division{
		ID:       "div1",
		Name:     "Division 1",
		MinElo:   1000,
		Category: "both",
	}

	div2 := &division.Division{
		ID:       "div2",
		Name:     "Division 2",
		MinElo:   0,
		Category: "both",
	}
	maxElo2 := int16(999)
	div2.MaxElo = &maxElo2

	p1 := &player.Player{ID: "p1", FirstName: "P1", SinglesElo: 1100}
	p2 := &player.Player{ID: "p2", FirstName: "P2", SinglesElo: 1100}
	p3 := &player.Player{ID: "p3", FirstName: "P3", SinglesElo: 1100}

	p4 := &player.Player{ID: "p4", FirstName: "P4", SinglesElo: 500}
	p5 := &player.Player{ID: "p5", FirstName: "P5", SinglesElo: 500}
	p6 := &player.Player{ID: "p6", FirstName: "P6", SinglesElo: 500}

	players := []*player.Player{p1, p2, p3, p4, p5, p6}

	trn := &event.Event{
		ID:           "t1",
		Format:       "groups_elimination",
		Type:         "singles",
		Participants: players,
		DivisionGroupPassCounts: map[string]int{
			"div1": 2,
			"div2": 3,
		},
		Groups: []event.Group{
			{
				ID:      "g1",
				Name:    "Division 1 - Group A",
				Players: []*player.Player{p1, p2, p3},
			},
			{
				ID:      "g2",
				Name:    "Division 2 - Group A",
				Players: []*player.Player{p4, p5, p6},
			},
		},
		Matches: []event.Match{},
	}

	// We need matches to make the groups "finished"
	// Match p1 vs p2, p2 vs p3, p1 vs p3
	addMatch := func(a, b *player.Player, scoreA, scoreB int, divID string) {
		m := event.Match{
			TeamA:      []*player.Player{a},
			TeamB:      []*player.Player{b},
			Status:     "finished",
			Stage:      "group",
			DivisionID: divID,
			Sets: []event.MatchSet{
				{ScoreA: scoreA, ScoreB: scoreB},
			},
		}
		if scoreA > scoreB {
			m.WinnerTeam = "A"
		} else {
			m.WinnerTeam = "B"
		}
		trn.Matches = append(trn.Matches, m)
	}

	addMatch(p1, p2, 11, 0, "div1")
	addMatch(p2, p3, 11, 0, "div1")
	addMatch(p1, p3, 11, 0, "div1")

	addMatch(p4, p5, 11, 0, "div2")
	addMatch(p5, p6, 11, 0, "div2")
	addMatch(p4, p6, 11, 0, "div2")

	vm := bracket.BuildBracket(trn, []*division.Division{div1, div2}, tmap)

	if len(vm.Divisions) != 2 {
		t.Fatalf("expected 2 divisions, got %d", len(vm.Divisions))
	}

	// Validate Division 1
	var div1View *bracket.Division
	var div2View *bracket.Division
	for i := range vm.Divisions {
		if vm.Divisions[i].ID == "div1" {
			div1View = &vm.Divisions[i]
		}
		if vm.Divisions[i].ID == "div2" {
			div2View = &vm.Divisions[i]
		}
	}

	if len(div1View.KnockoutAdvancing) != 2 {
		t.Errorf("expected 2 advancing in div1, got %d", len(div1View.KnockoutAdvancing))
	} else {
		// p1 and p2 should advance since they were top 2
		// Because of ITTF knockout seeds, p1 (seed 1) is at index 0, p2 (seed 2) is at index 1
		if div1View.KnockoutAdvancing[0].ID != "p1" || div1View.KnockoutAdvancing[1].ID != "p2" {
			t.Errorf("div1 advancing incorrect: %v", div1View.KnockoutAdvancing)
		}
	}

	if len(div2View.KnockoutAdvancing) != 3 {
		t.Errorf("expected 3 advancing in div2, got %d", len(div2View.KnockoutAdvancing))
	} else {
		// p4, p5, p6 should advance (all 3 in the group)
		// Bracket size 4, arrangement: 1, 4, 3, 2.
		// p4 is seed 1 (goes to index 0).
		// p5 is seed 2 (goes to bottom half).
		// p6 is seed 3 (goes to top half).
		// We just need to check they are all present.
		found := map[string]bool{}
		for _, p := range div2View.KnockoutAdvancing {
			found[p.ID] = true
		}
		if !found["p4"] || !found["p5"] || !found["p6"] {
			t.Errorf("div2 advancing incorrect, expected p4, p5, p6, got: %v", div2View.KnockoutAdvancing)
		}
	}

	if len(div1View.KnockoutRounds) == 0 {
		t.Errorf("expected KnockoutRounds for div1")
	} else {
		// Bracket size for 2 players should be 2 (1 round) + final
		// buildBracketRounds generates all rounds up to Final
		if len(div1View.KnockoutRounds[0].Matches) != 1 {
			t.Errorf("expected 1 match in div1 first round, got %d", len(div1View.KnockoutRounds[0].Matches))
		}
	}

	if len(div2View.KnockoutRounds) == 0 {
		t.Errorf("expected KnockoutRounds for div2")
	} else {
		// Bracket size for 3 players should be 4 (2 rounds) + final
		if len(div2View.KnockoutRounds[0].Matches) != 2 {
			t.Errorf("expected 2 matches in div2 first round, got %d", len(div2View.KnockoutRounds[0].Matches))
		}
	}
}
