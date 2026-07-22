package event

import (
	"table-tennis-backend/internal/domain/player"
	"testing"
)

func TestNewTeam(t *testing.T) {
	team, err := NewTeam("1", "t1", "Team A")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if team == nil {
		t.Fatalf("Expected Team to not be nil")
	}

	_, err = NewTeam("1", "t1", "")
	if err == nil || err.Error() != "team name cannot be empty" {
		t.Errorf("Expected team name empty error, got %v", err)
	}
}

func TestTeam_AverageElo(t *testing.T) {
	team, _ := NewTeam("1", "t1", "Team A")

	if team.AverageElo("singles") != 1000 {
		t.Errorf("Expected 1000 for empty team, got %v", team.AverageElo("singles"))
	}

	team.Players = append(team.Players, &player.Player{
		SinglesElo: 1200,
		DoublesElo: 1100,
	})
	team.Players = append(team.Players, &player.Player{
		SinglesElo: 1000,
		DoublesElo: 900,
	})

	if team.AverageElo("singles") != 1100 {
		t.Errorf("Expected 1100 for singles, got %v", team.AverageElo("singles"))
	}
	if team.AverageElo("doubles") != 1000 {
		t.Errorf("Expected 1000 for doubles, got %v", team.AverageElo("doubles"))
	}
}
