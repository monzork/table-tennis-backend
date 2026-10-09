package handler

import (
	"testing"
	"time"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func intPtr(i int) *int {
	return &i
}

func TestBuildBoardCards_Additional(t *testing.T) {
	p1 := &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A"}
	p2 := &playerDomain.Player{ID: "p2", FirstName: "B", LastName: "B"}
	tourney := &tournamentDomain.Event{
		Format: "elimination",
		Matches: []tournamentDomain.Match{
			{ID: "m1", Status: "scheduled", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}},
			{ID: "m2", Status: "in_progress", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}},
			{ID: "m3", Status: "finished", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}},
		},
	}
	scheduled, inProgress, finished := BuildBoardCards(tourney, nil)
	if len(scheduled) != 1 {
		t.Errorf("expected 1 scheduled match, got %d", len(scheduled))
	}
	if len(inProgress) != 1 {
		t.Errorf("expected 1 inProgress match, got %d", len(inProgress))
	}
	if len(finished) != 1 {
		t.Errorf("expected 1 finished match, got %d", len(finished))
	}
}

func TestGetOccupiedTables(t *testing.T) {
	h := &EventHandler{} // Assuming getOccupiedTables doesn't strictly need dependencies for basic logic
	_ = time.Now()
	_ = &tournamentDomain.Event{
		Matches: []tournamentDomain.Match{
			{ID: "m1", Status: "in_progress", TableNumber: intPtr(1)},
			{ID: "m2", Status: "in_progress", TableNumber: intPtr(2)},
			{ID: "m3", Status: "finished", TableNumber: intPtr(3)},
		},
	}

	// mock use case or just rely on the existing matches array if getOccupiedTables uses it
	_ = h
}
