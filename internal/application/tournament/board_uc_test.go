package tournament_test

import (
	"context"
	"testing"
	"time"

	"table-tennis-backend/internal/application/tournament"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
	eventDomain "table-tennis-backend/internal/domain/tournament"
)

func TestGetBoardDataUseCase_Execute(t *testing.T) {
	eventRepo := newMockEventRepo()
	divRepo := newMockDivisionRepo()

	uc := tournament.NewGetBoardDataUseCase(eventRepo, divRepo)

	ctx := context.Background()
	now := time.Now()

	// Setup basic event
	e := &eventDomain.Tournament{
		ID:        "e1",
		Name:      "Event 1",
		NumTables: 2,
	}

	// Setup some divisions
	div1, _ := divisionDomain.NewDivision("d1", "Div 1", 1, 1500, nil, "singles", "#ff0000")
	divRepo.divisions["d1"] = div1
	div2, _ := divisionDomain.NewDivision("d2", "Div 2", 1, 1500, nil, "singles", "#00ff00")
	divRepo.divisions["d2"] = div2

	p1 := &playerDomain.Player{ID: "p1", SinglesElo: 1000, Gender: "M"}
	p2 := &playerDomain.Player{ID: "p2", SinglesElo: 1100, Gender: "M"}

	m1 := tournamentDomain.Match{
		ID:     "m1",
		Status: "finished",
		TeamA:  []*playerDomain.Player{p1},
		TeamB:  []*playerDomain.Player{p2},
	}
	m1.UpdatedAt = &now

	t1, _ := tournamentDomain.NewTournament("t1", "Tourney 1", "singles", "single_elimination", "men", now, now.Add(24*time.Hour), nil, 1, []*playerDomain.Player{p1, p2}, false)
	t1.Matches = []tournamentDomain.Match{m1}

	e.Events = append(e.Events, t1)
	eventRepo.events["e1"] = e

	// test successful execution
	resE, resDivs, sched, inProg, fin, err := uc.Execute(ctx, "e1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resE.ID != "e1" {
		t.Errorf("expected e1")
	}
	if len(resDivs) != 2 {
		t.Errorf("expected 2 divs")
	}
	_ = sched
	_ = inProg
	_ = fin

	// Test repo error
	_, _, _, _, _, err = uc.Execute(ctx, "nonexistent")
	if err == nil {
		t.Errorf("expected error for nonexistent event")
	}
}

// Ensure coverage for complex sorting/allocation paths
func TestGetBoardDataUseCase_Tables(t *testing.T) {
	eventRepo := newMockEventRepo()
	divRepo := newMockDivisionRepo()
	uc := tournament.NewGetBoardDataUseCase(eventRepo, divRepo)
	ctx := context.Background()

	e := &eventDomain.Tournament{
		ID:        "e2",
		NumTables: 1, // fewer tables than divs
	}

	div1, _ := divisionDomain.NewDivision("d1", "Div 1", 1, 1500, nil, "singles", "#ff0000")
	divRepo.divisions["d1"] = div1
	div2, _ := divisionDomain.NewDivision("d2", "Div 2", 1, 1500, nil, "singles", "#00ff00")
	divRepo.divisions["d2"] = div2

	p1 := &playerDomain.Player{ID: "p1", SinglesElo: 1000, Gender: "M"}

	t1, _ := tournamentDomain.NewTournament("t1", "Tourney 1", "singles", "single_elimination", "men", time.Now(), time.Now(), nil, 1, []*playerDomain.Player{p1}, false)
	e.Events = append(e.Events, t1)
	eventRepo.events["e2"] = e

	// Should hit ratio calculation
	_, _, _, _, _, err := uc.Execute(ctx, "e2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGetBoardDataUseCase_Simulation exercises the greedy scheduling simulator
// in board_uc.go: multiple active divisions competing for fewer tables than
// divisions (fractional/remainder table allocation), an in-progress match
// (feeds availableTime/runningMatches), a finished match (feeds
// lastActivity), and a round_robin format so BuildBoardCards generates real
// virtual "scheduled" candidates for the queueing loop to consume.
func TestGetBoardDataUseCase_Simulation(t *testing.T) {
	eventRepo := newMockEventRepo()
	divRepo := newMockDivisionRepo()
	uc := tournament.NewGetBoardDataUseCase(eventRepo, divRepo)
	ctx := context.Background()
	now := time.Now()

	// 3 tables but 2 active divisions -> even allocation with 1 leftover
	// table distributed by fractional remainder.
	e := &eventDomain.Tournament{
		ID:        "e3",
		Name:      "Simulation Event",
		NumTables: 3,
	}

	maxEloLow := int16(1000)
	divLow, _ := divisionDomain.NewDivision("d_low", "Low", 1, 1, &maxEloLow, "both", "#ff0000")
	divHigh, _ := divisionDomain.NewDivision("d_high", "High", 2, 1001, nil, "both", "#00ff00")
	divRepo.divisions["d_low"] = divLow
	divRepo.divisions["d_high"] = divHigh

	p1 := &playerDomain.Player{ID: "p1", FirstName: "P", LastName: "1", SinglesElo: 600, Gender: "M"}
	p2 := &playerDomain.Player{ID: "p2", FirstName: "P", LastName: "2", SinglesElo: 700, Gender: "M"}
	p3 := &playerDomain.Player{ID: "p3", FirstName: "P", LastName: "3", SinglesElo: 1100, Gender: "M"}
	p4 := &playerDomain.Player{ID: "p4", FirstName: "P", LastName: "4", SinglesElo: 1200, Gender: "M"}
	p5 := &playerDomain.Player{ID: "p5", FirstName: "P", LastName: "5", SinglesElo: 1300, Gender: "M"}

	players := []*playerDomain.Player{p1, p2, p3, p4, p5}

	t1, _ := tournamentDomain.NewTournament("t1", "RR Tourney", "singles", "round_robin", "men", now, now.Add(24*time.Hour), nil, 1, players, false)

	// A finished match feeds the lastActivity map.
	finishedUpdatedAt := now
	finished := tournamentDomain.Match{
		ID:        "m_finished",
		Status:    "finished",
		TeamA:     []*playerDomain.Player{p1},
		TeamB:     []*playerDomain.Player{p2},
		UpdatedAt: &finishedUpdatedAt,
	}

	// An in-progress match with a table assigned feeds inProgress/availableTime/runningMatches.
	tableNum := 1
	inProgressMatch := tournamentDomain.Match{
		ID:          "m_inprogress",
		Status:      "in_progress",
		TeamA:       []*playerDomain.Player{p3},
		TeamB:       []*playerDomain.Player{p4},
		TableNumber: &tableNum,
	}

	t1.Matches = []tournamentDomain.Match{finished, inProgressMatch}

	e.Events = append(e.Events, t1)
	eventRepo.events["e3"] = e

	resE, resDivs, sched, inProg, fin, err := uc.Execute(ctx, "e3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resE == nil || resE.ID != "e3" {
		t.Errorf("expected event e3, got %+v", resE)
	}
	if len(resDivs) != 2 {
		t.Errorf("expected 2 divisions, got %d", len(resDivs))
	}
	if len(fin) != 1 {
		t.Errorf("expected 1 finished card, got %d", len(fin))
	}
	if len(inProg) != 1 {
		t.Errorf("expected 1 in-progress card, got %d", len(inProg))
	}
	// round_robin over 3+2 players (minus the already-played/in-progress
	// pairs) should still leave virtual candidate matches queued.
	if len(sched) == 0 {
		t.Error("expected queued scheduled cards from the round-robin virtual matches")
	}
	for i, c := range sched {
		if c.QueuePosition != i+1 {
			t.Errorf("expected queue position %d, got %d", i+1, c.QueuePosition)
		}
	}
}
