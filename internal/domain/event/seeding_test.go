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
	ev := &Event{Format: "elimination"}
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
}

func TestDivisionSeeder_AssignGroups(t *testing.T) {
	idgen.Register(dummyGen{})
	seeder := &DivisionSeeder{
		Divisions: []DivisionSeeding{
			{ID: "d1", Name: "Div 1", MinElo: 1400, MaxElo: nil},
		},
	}

	ev := &Event{Format: "elimination"}
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error")
	}

	ev.Format = "groups_elimination"
	p1 := &player.Player{ID: "1", FirstName: "A", SinglesElo: 1500}
	p2 := &player.Player{ID: "2", FirstName: "B", SinglesElo: 1000}
	ev.Participants = []*player.Player{p1, p2}

	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error")
	}

	if len(ev.Groups) != 2 {
		t.Fatalf("Expected 2 groups (Div 1 and Unclassified), got %d", len(ev.Groups))
	}
}

func TestDivisionSeeder_AssignGroups_UnsupportedFormat(t *testing.T) {
	seeder := &DivisionSeeder{}
	ev := &Event{Format: "elimination", Groups: []Group{{ID: "stale"}}}
	// "elimination" is a supported format for DivisionSeeder, so use a truly
	// unsupported one to hit the early-return branch that resets Groups.
	ev.Format = "unsupported"
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(ev.Groups) != 0 {
		t.Errorf("Expected Groups reset to empty, got %d", len(ev.Groups))
	}
}

func TestDivisionSeeder_AssignGroups_SkipEloOpenBracket(t *testing.T) {
	idgen.Register(dummyGen{})
	seeder := &DivisionSeeder{
		Divisions: []DivisionSeeding{{ID: "d1", Name: "Div 1", MinElo: 0, MaxElo: nil}},
	}
	p1 := &player.Player{ID: "1", SinglesElo: 1500}
	p2 := &player.Player{ID: "2", SinglesElo: 1000}
	ev := &Event{
		Format:       "groups_elimination",
		SkipElo:      true,
		Participants: []*player.Player{p1, p2},
	}
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(ev.Groups) != 1 {
		t.Fatalf("Expected 1 open-bracket group, got %d", len(ev.Groups))
	}
}

func TestDivisionSeeder_AssignGroups_SingleDivisionMultipleBrackets(t *testing.T) {
	idgen.Register(dummyGen{})
	seeder := &DivisionSeeder{
		Divisions: []DivisionSeeding{{ID: "d1", Name: "Div 1", MinElo: 0, MaxElo: nil}},
	}
	p1 := &player.Player{ID: "1", SinglesElo: 1500}
	ev := &Event{
		Format:       "single_division_multiple_brackets",
		Participants: []*player.Player{p1},
	}
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(ev.Groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(ev.Groups))
	}
}

func TestDivisionSeeder_AssignGroups_MaxEloBound(t *testing.T) {
	idgen.Register(dummyGen{})
	maxElo := int16(1450)
	seeder := &DivisionSeeder{
		Divisions: []DivisionSeeding{
			{ID: "d1", Name: "Div 1", MinElo: 1400, MaxElo: &maxElo},
		},
	}
	p1 := &player.Player{ID: "1", SinglesElo: 1500} // above max, goes unclassified
	p2 := &player.Player{ID: "2", SinglesElo: 1420} // within bounds
	ev := &Event{
		Format:       "groups_elimination",
		Participants: []*player.Player{p1, p2},
	}
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(ev.Groups) != 2 {
		t.Fatalf("Expected 2 groups (Div 1 and Unclassified), got %d", len(ev.Groups))
	}
}

func TestDivisionSeeder_AssignGroups_DivisionFormatOverrides(t *testing.T) {
	idgen.Register(dummyGen{})
	seeder := &DivisionSeeder{
		Divisions: []DivisionSeeding{
			{ID: "d1", Name: "Round Robin Div", MinElo: 1000, MaxElo: nil},
		},
	}
	p1 := &player.Player{ID: "1", SinglesElo: 1500}
	p2 := &player.Player{ID: "2", SinglesElo: 1400}
	ev := &Event{
		Format:       "groups_elimination",
		Participants: []*player.Player{p1, p2},
		DivisionConfigs: map[string]DivisionConfig{
			"d1": {Format: "round_robin"},
		},
	}
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(ev.Groups) != 1 {
		t.Fatalf("Expected 1 round-robin group for division, got %d", len(ev.Groups))
	}

	// Now try elimination format override
	ev.Groups = nil
	ev.DivisionConfigs["d1"] = DivisionConfig{Format: "elimination"}
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(ev.Groups) != 1 {
		t.Fatalf("Expected 1 bracket-draw group for division, got %d", len(ev.Groups))
	}
}

func TestDivisionSeeder_AssignGroups_ExplicitGroupCount(t *testing.T) {
	idgen.Register(dummyGen{})
	seeder := &DivisionSeeder{
		Divisions: []DivisionSeeding{
			{ID: "d1", Name: "Div 1", MinElo: 0, MaxElo: nil},
		},
	}
	players := []*player.Player{}
	for i := 0; i < 8; i++ {
		players = append(players, &player.Player{ID: string(rune('a' + i)), SinglesElo: int16(1500 - i*10)})
	}
	ev := &Event{
		Format:       "groups_elimination",
		Participants: players,
		DivisionConfigs: map[string]DivisionConfig{
			"d1": {GroupCount: 2},
		},
	}
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(ev.Groups) != 2 {
		t.Fatalf("Expected explicit group count of 2, got %d", len(ev.Groups))
	}
}

func TestDivisionSeeder_AssignGroups_TeamsAndDoublesElo(t *testing.T) {
	idgen.Register(dummyGen{})
	seeder := &DivisionSeeder{
		Divisions: []DivisionSeeding{{ID: "d1", Name: "Div 1", MinElo: 0, MaxElo: nil}},
	}
	team1, _ := NewTeam("t1", "tourn1", "Team One")
	team1.Players = []*player.Player{{DoublesElo: 1500}}
	team2, _ := NewTeam("t2", "tourn1", "Team Two")
	team2.Players = []*player.Player{{DoublesElo: 1400}}

	ev := &Event{
		Format: "groups_elimination",
		Type:   "doubles",
		Teams:  []*Team{team1, team2},
	}
	if err := seeder.AssignGroups(ev); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(ev.Groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(ev.Groups))
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
