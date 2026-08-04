package handler

import (
	"testing"

	"table-tennis-backend/internal/domain/division"
	domainEvent "table-tennis-backend/internal/domain/event"
)

func TestEventDivisionName(t *testing.T) {
	cases := map[string]string{
		"Ranking - Men's Singles (1st Division)": "1st Division",
		"Ranking - Women's Singles":              "",
	}
	for name, want := range cases {
		if got := EventDivisionName(name); got != want {
			t.Errorf("EventDivisionName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestSortEventsForPublicDisplay(t *testing.T) {
	divisions := []*division.Division{
		{Name: "1st Division", DisplayOrder: 1},
		{Name: "2nd Division", DisplayOrder: 2},
	}
	events := []*domainEvent.Event{
		{Name: "T - Men's Singles (2nd Division)", EventCategory: "men"},
		{Name: "T - Mixed Doubles", EventCategory: "mixed"},
		{Name: "T - Women's Singles (1st Division)", EventCategory: "women"},
		{Name: "T - Men's Singles (1st Division)", EventCategory: "men"},
		{Name: "T - Women's Singles (2nd Division)", EventCategory: "women"},
	}

	sortEventsForPublicDisplay(events, divisions)

	want := []string{
		"T - Women's Singles (1st Division)",
		"T - Women's Singles (2nd Division)",
		"T - Men's Singles (1st Division)",
		"T - Men's Singles (2nd Division)",
		"T - Mixed Doubles",
	}
	for i, e := range events {
		if e.Name != want[i] {
			t.Errorf("position %d = %q, want %q", i, e.Name, want[i])
		}
	}
}
