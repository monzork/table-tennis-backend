package event

import (
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
	"testing"
)

type dummyGen struct{}

func (d dummyGen) Generate() string { return "id" }

func TestOpenBracketSnakeSeeder_AssignGroups(t *testing.T) {
	idgen.Register(dummyGen{})
	seeder := &OpenBracketSnakeSeeder{}

	// Format not matching
	ev := &Event{Format: "unsupported"}
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error")
	}

	// No participants
	ev.Format = "groups_elimination"
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error")
	}

	p1 := &player.Player{ID: "1", FirstName: "A", SinglesElo: 1500}
	p2 := &player.Player{ID: "2", FirstName: "B", SinglesElo: 1400}
	p3 := &player.Player{ID: "3", FirstName: "C", SinglesElo: 1300}
	p4 := &player.Player{ID: "4", FirstName: "D", SinglesElo: 1200}

	ev.Participants = []*player.Player{p4, p3, p2, p1}

	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error")
	}
	if len(ev.Groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(ev.Groups))
	}

	// Round robin single group
	ev.Format = "round_robin"
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error")
	}
	if len(ev.Groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(ev.Groups))
	}

	// Elimination: single bracket-draw group
	ev.Format = "elimination"
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error")
	}
	if len(ev.Groups) != 1 || ev.Groups[0].Name != "Bracket Draw" {
		t.Fatalf("Expected 1 Bracket Draw group, got %+v", ev.Groups)
	}

	// single_division_multiple_brackets: numbered groups via n/4 heuristic
	ev.Format = "single_division_multiple_brackets"
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error")
	}
	if len(ev.Groups) != 1 {
		t.Fatalf("Expected 1 group for 4 participants, got %d", len(ev.Groups))
	}
}

func TestOpenBracketSnakeSeeder_AssignGroups_Teams(t *testing.T) {
	idgen.Register(dummyGen{})
	seeder := &OpenBracketSnakeSeeder{}
	team1, _ := NewTeam("t1", "tourn1", "Team One")
	team1.Players = []*player.Player{{SinglesElo: 1500}}
	team2, _ := NewTeam("t2", "tourn1", "Team Two")
	team2.Players = []*player.Player{{SinglesElo: 1400}}

	ev := &Event{
		Format: "groups_elimination",
		Type:   "teams",
		Teams:  []*Team{team1, team2},
	}
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(ev.Groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(ev.Groups))
	}
}

func TestOpenBracketSnakeSeeder_AssignGroups_GroupCountOverride(t *testing.T) {
	idgen.Register(dummyGen{})
	seeder := &OpenBracketSnakeSeeder{}
	players := []*player.Player{}
	for i := 0; i < 8; i++ {
		players = append(players, &player.Player{ID: string(rune('a' + i)), SinglesElo: int16(1500 - i*10)})
	}

	// Explicit override takes precedence over the groups-of-4 default.
	ev := &Event{Format: "groups_elimination", Participants: players, GroupCount: 3}
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(ev.Groups) != 3 {
		t.Fatalf("Expected 3 groups from override, got %d", len(ev.Groups))
	}
	total := 0
	for _, g := range ev.Groups {
		total += len(g.Players)
	}
	if total != 8 {
		t.Errorf("Expected all 8 players distributed, got %d", total)
	}

	// Override larger than the participant count is clamped to participant count.
	ev2 := &Event{Format: "groups_elimination", Participants: players, GroupCount: 20}
	if err := seeder.AssignGroups(ev2); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(ev2.Groups) != len(players) {
		t.Fatalf("Expected %d groups (clamped), got %d", len(players), len(ev2.Groups))
	}
}

func TestOpenBracketSnakeSeeder_AssignGroups_SnakeRows(t *testing.T) {
	idgen.Register(dummyGen{})
	seeder := &OpenBracketSnakeSeeder{}
	players := []*player.Player{}
	for i := 0; i < 8; i++ {
		players = append(players, &player.Player{ID: string(rune('a' + i)), SinglesElo: int16(1500 - i*10)})
	}
	ev := &Event{Format: "groups_elimination", Participants: players}
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(ev.Groups) != 2 {
		t.Fatalf("Expected 2 groups for 8 players, got %d", len(ev.Groups))
	}
	total := 0
	for _, g := range ev.Groups {
		total += len(g.Players)
	}
	if total != 8 {
		t.Errorf("Expected all 8 players distributed, got %d", total)
	}
}
