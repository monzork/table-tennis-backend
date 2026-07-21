package event

import (
	"table-tennis-backend/internal/domain/player"
	"testing"
)

func TestGetTeamDisplayName(t *testing.T) {
	if GetTeamDisplayName(nil, "singles") != "N/A" {
		t.Errorf("Expected N/A")
	}

	p1 := &player.Player{FirstName: "John", LastName: "Doe"}
	if GetTeamDisplayName([]*player.Player{p1}, "teams") != "John" {
		t.Errorf("Expected John")
	}

	if GetTeamDisplayName([]*player.Player{p1}, "singles") != "John Doe" {
		t.Errorf("Expected John Doe")
	}

	pNoLast := &player.Player{FirstName: "Jane"}
	if GetTeamDisplayName([]*player.Player{pNoLast}, "singles") != "Jane" {
		t.Errorf("Expected Jane")
	}

	p2 := &player.Player{FirstName: "Bob", LastName: "Smith"}
	if GetTeamDisplayName([]*player.Player{p1, p2}, "doubles") != "John Doe / Bob Smith" {
		t.Errorf("Expected John Doe / Bob Smith")
	}

	p3 := &player.Player{FirstName: "Alice", LastName: "Wonder"}
	if GetTeamDisplayName([]*player.Player{p1, p2, p3}, "teams") != "John" {
		t.Errorf("Expected John")
	}
	if GetTeamDisplayName([]*player.Player{p1, p2, p3}, "doubles") != "John Doe" { // fallback
		t.Errorf("Expected John Doe")
	}
}

func TestGetTournamentPlaces_NotFinished(t *testing.T) {
	ev := &Event{Status: "in_progress"}
	first, second, third := GetTournamentPlaces(ev)
	if first != "" || second != "" || third != "" {
		t.Errorf("Expected empty places for unfinished event")
	}
}

func TestGetTournamentPlaces_Elimination(t *testing.T) {
	ev := &Event{
		Status: "finished",
		Format: "elimination",
		Type:   "singles",
		Matches: []Match{
			{
				Stage:      "final",
				Status:     "finished",
				WinnerTeam: "A",
				TeamA:      []*player.Player{{FirstName: "John", LastName: "Doe"}},
				TeamB:      []*player.Player{{FirstName: "Jane", LastName: "Doe"}},
			},
			{
				Stage:      "semifinal",
				Status:     "finished",
				WinnerTeam: "A",
				TeamA:      []*player.Player{{FirstName: "Bob", LastName: "Smith"}},
				TeamB:      []*player.Player{{FirstName: "Alice", LastName: "Wonder"}}, // Alice is semi loser
			},
			{
				Stage:      "semifinal",
				Status:     "finished",
				WinnerTeam: "B",
				TeamA:      []*player.Player{{FirstName: "Charlie", LastName: "Brown"}}, // Charlie is semi loser
				TeamB:      []*player.Player{{FirstName: "John", LastName: "Doe"}},
			},
		},
	}

	first, second, third := GetTournamentPlaces(ev)
	if first != "John Doe" {
		t.Errorf("Expected first to be John Doe, got %v", first)
	}
	if second != "Jane Doe" {
		t.Errorf("Expected second to be Jane Doe, got %v", second)
	}
	if third != "Alice Wonder & Charlie Brown" {
		t.Errorf("Expected third to be Alice Wonder & Charlie Brown, got %v", third)
	}
}

func TestGetTournamentPlaces_EliminationNoFinalFallsBackToWinnerName(t *testing.T) {
	ev := &Event{
		Status:     "finished",
		Format:     "elimination",
		Type:       "singles",
		WinnerName: "John Doe",
	}
	first, second, third := GetTournamentPlaces(ev)
	if first != "John Doe" {
		t.Errorf("Expected fallback WinnerName John Doe, got %v", first)
	}
	if second != "" || third != "" {
		t.Errorf("Expected empty second/third, got %v/%v", second, third)
	}
}

func TestGetTournamentPlaces_RoundRobin_Singles(t *testing.T) {
	p1 := &player.Player{ID: "p1", FirstName: "John", LastName: "Doe", SinglesElo: 1500}
	p2 := &player.Player{ID: "p2", FirstName: "Jane", LastName: "Doe", SinglesElo: 1400}
	p3 := &player.Player{ID: "p3", FirstName: "Bob", LastName: "Smith", SinglesElo: 1300}

	ev := &Event{
		Status:       "finished",
		Format:       "round_robin",
		Type:         "singles",
		Participants: []*player.Player{p1, p2, p3},
		Matches: []Match{
			{Status: "finished", Stage: "group", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}, Sets: []MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}, {Number: 2, ScoreA: 11, ScoreB: 5}, {Number: 3, ScoreA: 11, ScoreB: 5}}},
			{Status: "finished", Stage: "group", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p3}, Sets: []MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}, {Number: 2, ScoreA: 11, ScoreB: 5}, {Number: 3, ScoreA: 11, ScoreB: 5}}},
			{Status: "finished", Stage: "group", WinnerTeam: "A", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p3}, Sets: []MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}, {Number: 2, ScoreA: 11, ScoreB: 5}, {Number: 3, ScoreA: 11, ScoreB: 5}}},
		},
	}

	first, second, third := GetTournamentPlaces(ev)
	if first != "John Doe" {
		t.Errorf("Expected first to be John Doe, got %v", first)
	}
	if second != "Jane Doe" {
		t.Errorf("Expected second to be Jane Doe, got %v", second)
	}
	if third != "Bob Smith" {
		t.Errorf("Expected third to be Bob Smith, got %v", third)
	}
}

func TestGetTournamentPlaces_RoundRobin_Teams(t *testing.T) {
	team1, _ := NewTeam("t1", "tourn1", "Team One")
	team1.Players = []*player.Player{{SinglesElo: 1500}}
	team2, _ := NewTeam("t2", "tourn1", "Team Two")
	team2.Players = []*player.Player{{SinglesElo: 1400}}

	ev := &Event{
		Status: "finished",
		Format: "round_robin",
		Type:   "teams",
		Teams:  []*Team{team1, team2},
		Matches: []Match{
			{Status: "finished", Stage: "group", WinnerTeam: "A",
				TeamA: []*player.Player{{ID: "t1"}}, TeamB: []*player.Player{{ID: "t2"}},
				Sets: []MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}, {Number: 2, ScoreA: 11, ScoreB: 5}, {Number: 3, ScoreA: 11, ScoreB: 5}}},
		},
	}

	first, _, _ := GetTournamentPlaces(ev)
	if first != "Team One" {
		t.Errorf("Expected first to be Team One, got %v", first)
	}
}

func TestGetTournamentPlaces_RoundRobin_NoParticipants(t *testing.T) {
	ev := &Event{Status: "finished", Format: "round_robin", Type: "singles"}
	first, second, third := GetTournamentPlaces(ev)
	if first != "" || second != "" || third != "" {
		t.Errorf("Expected all empty for no participants, got %v/%v/%v", first, second, third)
	}
}

func TestGetTournamentPlaces_UnknownFormat(t *testing.T) {
	ev := &Event{Status: "finished", Format: "unknown"}
	first, second, third := GetTournamentPlaces(ev)
	if first != "" || second != "" || third != "" {
		t.Errorf("Expected all empty for unknown format, got %v/%v/%v", first, second, third)
	}
}
