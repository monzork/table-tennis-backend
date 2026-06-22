package tournament

import (
	"testing"

	"github.com/google/uuid"
	"table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

func TestGetTournamentPlaces_NotFinished(t *testing.T) {
	tourney := &tournamentDomain.Tournament{
		ID:     uuid.New(),
		Name:   "Unfinished Tourney",
		Format: "elimination",
		Status: "in_progress",
	}

	first, second, third := getTournamentPlaces(tourney)
	if first != "" || second != "" || third != "" {
		t.Errorf("expected empty places for unfinished tournament, got: 1st='%s', 2nd='%s', 3rd='%s'", first, second, third)
	}
}

func TestGetTournamentPlaces_Elimination(t *testing.T) {
	p1 := &player.Player{ID: uuid.New(), FirstName: "Alice", LastName: "Smith"}
	p2 := &player.Player{ID: uuid.New(), FirstName: "Bob", LastName: "Jones"}
	p3 := &player.Player{ID: uuid.New(), FirstName: "Charlie", LastName: "Brown"}
	p4 := &player.Player{ID: uuid.New(), FirstName: "Diana", LastName: "Prince"}

	tourney := &tournamentDomain.Tournament{
		ID:           uuid.New(),
		Name:         "Elimination Cup",
		Type:         "singles",
		Format:       "elimination",
		Status:       "finished",
		Participants: []*player.Player{p1, p2, p3, p4},
		Matches: []tournamentDomain.Match{
			// Semifinals
			{
				ID:         uuid.New(),
				Stage:      "semifinal",
				Status:     "finished",
				TeamA:      []*player.Player{p1},
				TeamB:      []*player.Player{p3},
				WinnerTeam: "A", // Alice beats Charlie
				Sets:       []tournamentDomain.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}},
			},
			{
				ID:         uuid.New(),
				Stage:      "semifinal",
				Status:     "finished",
				TeamA:      []*player.Player{p2},
				TeamB:      []*player.Player{p4},
				WinnerTeam: "A", // Bob beats Diana
				Sets:       []tournamentDomain.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 8}},
			},
			// Final
			{
				ID:         uuid.New(),
				Stage:      "final",
				Status:     "finished",
				TeamA:      []*player.Player{p1},
				TeamB:      []*player.Player{p2},
				WinnerTeam: "B", // Bob beats Alice
				Sets:       []tournamentDomain.MatchSet{{Number: 1, ScoreA: 9, ScoreB: 11}},
			},
		},
	}

	first, second, third := getTournamentPlaces(tourney)

	if first != "Bob Jones" {
		t.Errorf("expected 1st place to be 'Bob Jones', got '%s'", first)
	}
	if second != "Alice Smith" {
		t.Errorf("expected 2nd place to be 'Alice Smith', got '%s'", second)
	}
	// Joint 3rd place should contain Charlie Brown and Diana Prince (semifinal losers)
	if !stringsContains(third, "Charlie Brown") || !stringsContains(third, "Diana Prince") {
		t.Errorf("expected 3rd place to contain 'Charlie Brown' and 'Diana Prince', got '%s'", third)
	}
}

func TestGetTournamentPlaces_RoundRobin(t *testing.T) {
	p1 := &player.Player{ID: uuid.New(), FirstName: "Alice", LastName: "Smith", SinglesElo: 1000}
	p2 := &player.Player{ID: uuid.New(), FirstName: "Bob", LastName: "Jones", SinglesElo: 900}
	p3 := &player.Player{ID: uuid.New(), FirstName: "Charlie", LastName: "Brown", SinglesElo: 800}

	tourney := &tournamentDomain.Tournament{
		ID:           uuid.New(),
		Name:         "Round Robin League",
		Type:         "singles",
		Format:       "round_robin",
		Status:       "finished",
		Participants: []*player.Player{p1, p2, p3},
		Matches: []tournamentDomain.Match{
			{
				ID:         uuid.New(),
				Stage:      "group",
				Status:     "finished",
				TeamA:      []*player.Player{p1},
				TeamB:      []*player.Player{p2},
				WinnerTeam: "A", // Alice beats Bob
				Sets:       []tournamentDomain.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}},
			},
			{
				ID:         uuid.New(),
				Stage:      "group",
				Status:     "finished",
				TeamA:      []*player.Player{p2},
				TeamB:      []*player.Player{p3},
				WinnerTeam: "A", // Bob beats Charlie
				Sets:       []tournamentDomain.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}},
			},
			{
				ID:         uuid.New(),
				Stage:      "group",
				Status:     "finished",
				TeamA:      []*player.Player{p1},
				TeamB:      []*player.Player{p3},
				WinnerTeam: "A", // Alice beats Charlie
				Sets:       []tournamentDomain.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}},
			},
		},
	}

	first, second, third := getTournamentPlaces(tourney)

	if first != "Alice Smith" {
		t.Errorf("expected 1st place to be 'Alice Smith', got '%s'", first)
	}
	if second != "Bob Jones" {
		t.Errorf("expected 2nd place to be 'Bob Jones', got '%s'", second)
	}
	if third != "Charlie Brown" {
		t.Errorf("expected 3rd place to be 'Charlie Brown', got '%s'", third)
	}
}

func stringsContains(s, substr string) bool {
	// Simple helper for test to avoid dependency on "strings" package if not imported,
	// wait, we can just import strings.
	return len(s) >= len(substr) && (s == substr || s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || stringsContainsInner(s, substr))
}

func stringsContainsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
