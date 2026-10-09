package handler

import (
	"testing"

	"table-tennis-backend/internal/application/event"
	tournamentDomain "table-tennis-backend/internal/domain/event"
)

func TestFilterBoardCards(t *testing.T) {
	cards := []event.BoardCard{
		{P1Id: "p1", P2Id: "p2", PlayerAName: "Alice", PlayerBName: "Bob", DivisionName: "Div1"},
		{P1Id: "p3", P2Id: "p4", PlayerAName: "Charlie", PlayerBName: "David", DivisionName: "Div2"},
	}

	filtered := FilterBoardCards(cards, "alice", []string{})
	if len(filtered) != 1 || filtered[0].PlayerAName != "Alice" {
		t.Errorf("expected 1 match for Alice, got %d", len(filtered))
	}

	filteredDiv := FilterBoardCards(cards, "", []string{"Div2"})
	if len(filteredDiv) != 1 || filteredDiv[0].DivisionName != "Div2" {
		t.Errorf("expected 1 match for Div2, got %d", len(filteredDiv))
	}
}

func TestBuildBoardCards(t *testing.T) {
	tourney := &tournamentDomain.Event{
		Format: "elimination",
	}
	scheduled, inProgress, finished := BuildBoardCards(tourney, nil)
	if len(scheduled) != 0 || len(inProgress) != 0 || len(finished) != 0 {
		t.Errorf("expected empty slices for empty event")
	}
}
