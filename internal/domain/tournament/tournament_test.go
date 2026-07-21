package tournament_test

import (
	"testing"
	"time"

	"table-tennis-backend/internal/domain/tournament"
)

func TestNewEvent_Success(t *testing.T) {
	start := time.Now()
	end := start.Add(48 * time.Hour)
	divIDs := []string{"div1", "div2"}

	tr, err := tournament.NewEvent("t-1", "Summer Open", divIDs, false, start, end)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if tr.ID != "t-1" {
		t.Errorf("expected ID 't-1', got '%s'", tr.ID)
	}
	if tr.Name != "Summer Open" {
		t.Errorf("expected Name 'Summer Open', got '%s'", tr.Name)
	}
	if len(tr.DivisionIDs) != 2 || tr.DivisionIDs[0] != "div1" || tr.DivisionIDs[1] != "div2" {
		t.Errorf("expected DivisionIDs ['div1', 'div2'], got %v", tr.DivisionIDs)
	}
	if tr.SkipElo != false {
		t.Errorf("expected SkipElo false, got %v", tr.SkipElo)
	}
	if !tr.StartDate.Equal(start) {
		t.Errorf("expected StartDate %v, got %v", start, tr.StartDate)
	}
	if !tr.EndDate.Equal(end) {
		t.Errorf("expected EndDate %v, got %v", end, tr.EndDate)
	}
	if tr.NumTables != 4 {
		t.Errorf("expected default NumTables 4, got %d", tr.NumTables)
	}
	if tr.Events == nil || len(tr.Events) != 0 {
		t.Errorf("expected empty Events slice, got %v", tr.Events)
	}
}

func TestNewEvent_SkipEloTrueWithNoDivisions(t *testing.T) {
	start := time.Now()
	end := start.Add(24 * time.Hour)

	tr, err := tournament.NewEvent("t-2", "Casual Event", nil, true, start, end)
	if err != nil {
		t.Fatalf("expected no error when SkipElo is true and divisionIDs is empty, got %v", err)
	}

	if tr.SkipElo != true {
		t.Errorf("expected SkipElo true, got %v", tr.SkipElo)
	}
}

func TestNewEvent_InvalidName(t *testing.T) {
	start := time.Now()
	end := start.Add(24 * time.Hour)

	_, err := tournament.NewEvent("t-1", "", []string{"div1"}, false, start, end)
	if err != tournament.ErrInvalidEventName {
		t.Fatalf("expected ErrInvalidEventName, got %v", err)
	}
}

func TestNewEvent_InvalidDivisionIDs(t *testing.T) {
	start := time.Now()
	end := start.Add(24 * time.Hour)

	// skipElo is false, divisionIDs is empty -> should fail
	_, err := tournament.NewEvent("t-1", "Test Tourn", []string{}, false, start, end)
	if err != tournament.ErrInvalidDivisionIDs {
		t.Fatalf("expected ErrInvalidDivisionIDs, got %v", err)
	}

	_, err = tournament.NewEvent("t-1", "Test Tourn", nil, false, start, end)
	if err != tournament.ErrInvalidDivisionIDs {
		t.Fatalf("expected ErrInvalidDivisionIDs, got %v", err)
	}
}

func TestNewEvent_InvalidEventDates(t *testing.T) {
	start := time.Now()
	end := start.Add(-1 * time.Hour) // end date before start date

	_, err := tournament.NewEvent("t-1", "Test Tourn", []string{"div1"}, false, start, end)
	if err != tournament.ErrInvalidEventDates {
		t.Fatalf("expected ErrInvalidEventDates, got %v", err)
	}
}
