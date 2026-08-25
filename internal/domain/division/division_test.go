package division_test

import (
	"testing"

	"table-tennis-backend/internal/domain/division"
)

func TestNewDivision_Success(t *testing.T) {
	maxElo := int16(2000)
	d, err := division.NewDivision("div-1", "Division A", 1, 1000, &maxElo, "singles", "#ff0000")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if d.ID != "div-1" {
		t.Errorf("expected ID 'div-1', got '%s'", d.ID)
	}
	if d.Name != "Division A" {
		t.Errorf("expected Name 'Division A', got '%s'", d.Name)
	}
	if d.DisplayOrder != 1 {
		t.Errorf("expected DisplayOrder 1, got %d", d.DisplayOrder)
	}
	if d.MinElo != 1000 {
		t.Errorf("expected MinElo 1000, got %d", d.MinElo)
	}
	if d.MaxElo == nil || *d.MaxElo != 2000 {
		t.Errorf("expected MaxElo 2000, got %v", d.MaxElo)
	}
	if d.Category != "singles" {
		t.Errorf("expected Category 'singles', got '%s'", d.Category)
	}
	if d.Color != "#ff0000" {
		t.Errorf("expected Color '#ff0000', got '%s'", d.Color)
	}
}

func TestNewDivision_Defaults(t *testing.T) {
	d, err := division.NewDivision("div-2", "Division B", 2, 500, nil, "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if d.Category != "both" {
		t.Errorf("expected default Category 'both', got '%s'", d.Category)
	}
	if d.Gender != "both" {
		t.Errorf("expected default Gender 'both', got '%s'", d.Gender)
	}
	if d.Color != "#ffffff" {
		t.Errorf("expected default Color '#ffffff', got '%s'", d.Color)
	}
	if d.MaxElo != nil {
		t.Errorf("expected MaxElo to be nil, got %v", d.MaxElo)
	}
}

func TestNewDivision_InvalidName(t *testing.T) {
	_, err := division.NewDivision("div-1", "", 1, 1000, nil, "singles", "#ffffff")
	if err != division.ErrInvalidName {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}

func TestNewDivision_InvalidEloRange(t *testing.T) {
	maxElo := int16(1000)
	// minElo (1000) >= maxElo (1000) -> should fail
	_, err := division.NewDivision("div-1", "Division A", 1, 1000, &maxElo, "singles", "#ffffff")
	if err != division.ErrInvalidEloRange {
		t.Fatalf("expected ErrInvalidEloRange, got %v", err)
	}

	maxElo2 := int16(800)
	// minElo (1000) > maxElo (800) -> should fail
	_, err = division.NewDivision("div-1", "Division A", 1, 1000, &maxElo2, "singles", "#ffffff")
	if err != division.ErrInvalidEloRange {
		t.Fatalf("expected ErrInvalidEloRange, got %v", err)
	}
}

func TestDivision_ContainsElo(t *testing.T) {
	maxElo := int16(1500)
	dWithMax, _ := division.NewDivision("d1", "Bounded", 1, 1000, &maxElo, "singles", "#000000")
	dNoMax, _ := division.NewDivision("d2", "Top", 2, 1500, nil, "singles", "#000000")

	tests := []struct {
		division *division.Division
		elo      int16
		expected bool
		label    string
	}{
		{dWithMax, 999, false, "below min Elo"},
		{dWithMax, 1000, true, "exactly min Elo (inclusive)"},
		{dWithMax, 1250, true, "within range"},
		{dWithMax, 1499, true, "just below max Elo"},
		{dWithMax, 1500, false, "exactly max Elo (exclusive)"},
		{dWithMax, 1600, false, "above max Elo"},
		{dNoMax, 1499, false, "below min Elo (unbounded max)"},
		{dNoMax, 1500, true, "exactly min Elo (unbounded max)"},
		{dNoMax, 2500, true, "high Elo (unbounded max)"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got := tt.division.ContainsElo(tt.elo)
			if got != tt.expected {
				t.Errorf("ContainsElo(%d) = %v, want %v", tt.elo, got, tt.expected)
			}
		})
	}
}

func TestDivision_MatchesGender(t *testing.T) {
	both, _ := division.NewDivision("d1", "Both", 1, 0, nil, "singles", "#000000")

	mens, _ := division.NewDivision("d2", "Men's 1st", 1, 1600, nil, "singles", "#000000")
	mens.Gender = "M"

	womens, _ := division.NewDivision("d3", "Women's 1st", 1, 900, nil, "singles", "#000000")
	womens.Gender = "F"

	unset := &division.Division{ID: "d4", Name: "Legacy", MinElo: 0}

	tests := []struct {
		division *division.Division
		gender   string
		expected bool
		label    string
	}{
		{both, "M", true, "both-gender division matches men"},
		{both, "F", true, "both-gender division matches women"},
		{mens, "M", true, "men's division matches men"},
		{mens, "F", false, "men's division does not match women"},
		{womens, "F", true, "women's division matches women"},
		{womens, "M", false, "women's division does not match men"},
		{unset, "M", true, "empty Gender (legacy fixture) matches men"},
		{unset, "F", true, "empty Gender (legacy fixture) matches women"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got := tt.division.MatchesGender(tt.gender)
			if got != tt.expected {
				t.Errorf("MatchesGender(%q) = %v, want %v", tt.gender, got, tt.expected)
			}
		})
	}
}

func TestOnlyGendered(t *testing.T) {
	both, _ := division.NewDivision("d1", "Both", 1, 0, nil, "singles", "#000000")

	mens, _ := division.NewDivision("d2", "Men's 1st", 1, 1600, nil, "singles", "#000000")
	mens.Gender = "M"

	womens, _ := division.NewDivision("d3", "Women's 1st", 1, 900, nil, "singles", "#000000")
	womens.Gender = "F"

	unset := &division.Division{ID: "d4", Name: "Legacy", MinElo: 0}

	got := division.OnlyGendered([]*division.Division{both, mens, womens, unset})

	if len(got) != 2 {
		t.Fatalf("expected only the 2 gender-specific divisions to survive, got %d: %+v", len(got), got)
	}
	if got[0].ID != "d2" || got[1].ID != "d3" {
		t.Errorf("expected [mens, womens] in order, got %+v", got)
	}
}
