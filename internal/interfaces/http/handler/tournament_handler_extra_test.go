package handler

import (
	"testing"

	"table-tennis-backend/internal/application/event"
	domainEvent "table-tennis-backend/internal/domain/event"
	eventDomain "table-tennis-backend/internal/domain/tournament"
)

func TestFilterEventBoardCards(t *testing.T) {
	cards := []event.BoardCard{
		{PlayerAName: "Alice", PlayerBName: "Bob", GroupName: "Final", DivisionName: "Div1", Category: "Men"},
		{PlayerAName: "Charlie", PlayerBName: "Dave", GroupName: "Semi", DivisionName: "Div2", Category: "Women"},
	}

	res := FilterEventBoardCards(cards, "alice", nil, nil)
	if len(res) != 1 {
		t.Errorf("Expected 1 card, got %d", len(res))
	}

	res2 := FilterEventBoardCards(cards, "", []string{"Div2"}, nil)
	if len(res2) != 1 {
		t.Errorf("Expected 1 card, got %d", len(res2))
	}

	res3 := FilterEventBoardCards(cards, "", nil, []string{"Men"})
	if len(res3) != 1 {
		t.Errorf("Expected 1 card, got %d", len(res3))
	}

	res4 := FilterEventBoardCards(cards, "", nil, nil)
	if len(res4) != 2 {
		t.Errorf("Expected no-op with empty filters to return all 2 cards, got %d", len(res4))
	}
}

func TestBuildEventTables(t *testing.T) {
	e := &eventDomain.Tournament{
		NumTables: 4,
	}
	tableNum := 2
	inProgress := []event.BoardCard{
		{TableNumber: &tableNum},
	}
	tables := buildEventTables(e, inProgress)
	if len(tables) != 4 {
		t.Fatalf("Expected 4 tables, got %d", len(tables))
	}
	if tables[0].IsUsed {
		t.Errorf("Expected table 1 to be unused")
	}
	if !tables[1].IsUsed {
		t.Errorf("Expected table 2 to be used")
	}

	t.Run("nil tournament returns no tables", func(t *testing.T) {
		if tbls := buildEventTables(nil, nil); tbls != nil {
			t.Errorf("expected nil tables for nil tournament, got %v", tbls)
		}
	})

	t.Run("child event with more tables than parent wins", func(t *testing.T) {
		eChild := &eventDomain.Tournament{
			NumTables: 2,
			Events: []*domainEvent.Event{
				{NumTables: 6},
			},
		}
		tbls := buildEventTables(eChild, nil)
		if len(tbls) != 6 {
			t.Errorf("expected 6 tables (child event NumTables wins), got %d", len(tbls))
		}
	})

	t.Run("zero tables returns empty", func(t *testing.T) {
		eZero := &eventDomain.Tournament{NumTables: 0}
		if tbls := buildEventTables(eZero, nil); len(tbls) != 0 {
			t.Errorf("expected 0 tables, got %d", len(tbls))
		}
	})
}
