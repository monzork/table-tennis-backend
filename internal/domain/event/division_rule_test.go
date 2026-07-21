package event

import (
	"testing"
)

func TestNewDivisionRule(t *testing.T) {
	dr, err := NewDivisionRule("t1", "d1", 5, 11, 2)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if dr == nil {
		t.Fatalf("Expected DivisionRule to not be nil")
	}

	if dr.ID != "t1-d1" {
		t.Errorf("Expected ID to be t1-d1, got %v", dr.ID)
	}

	_, err = NewDivisionRule("", "d1", 5, 11, 2)
	if err != ErrInvalidTournamentID {
		t.Errorf("Expected ErrInvalidTournamentID, got %v", err)
	}

	_, err = NewDivisionRule("t1", "", 5, 11, 2)
	if err != ErrInvalidDivisionID {
		t.Errorf("Expected ErrInvalidDivisionID, got %v", err)
	}

	_, err = NewDivisionRule("t1", "d1", 4, 11, 2)
	if err != ErrInvalidBestOf {
		t.Errorf("Expected ErrInvalidBestOf, got %v", err)
	}
	_, err = NewDivisionRule("t1", "d1", 2, 11, 2)
	if err != ErrInvalidBestOf {
		t.Errorf("Expected ErrInvalidBestOf, got %v", err)
	}

	_, err = NewDivisionRule("t1", "d1", 3, 0, 2)
	if err != ErrInvalidPointsToWin {
		t.Errorf("Expected ErrInvalidPointsToWin, got %v", err)
	}

	_, err = NewDivisionRule("t1", "d1", 3, 11, 0)
	if err != ErrInvalidPointsMargin {
		t.Errorf("Expected ErrInvalidPointsMargin, got %v", err)
	}

	// Test Error()
	if ErrInvalidTournamentID.Error() != "event ID is required" {
		t.Errorf("Expected error message 'event ID is required', got %v", ErrInvalidTournamentID.Error())
	}
}

func TestDivisionRule_ToStageRule(t *testing.T) {
	dr, _ := NewDivisionRule("t1", "d1", 5, 11, 2)
	sr := dr.ToStageRule()
	if sr.ID != "t1-d1" {
		t.Errorf("Expected ID to be t1-d1, got %v", sr.ID)
	}
	if sr.TournamentID != "t1" {
		t.Errorf("Expected TournamentID to be t1, got %v", sr.TournamentID)
	}
	if sr.Stage != "division_override" {
		t.Errorf("Expected Stage to be division_override, got %v", sr.Stage)
	}
	if sr.BestOf != 5 {
		t.Errorf("Expected BestOf to be 5, got %v", sr.BestOf)
	}
}
