package event

import (
	"table-tennis-backend/internal/domain/player"
	"testing"
)

func TestGetDivisionFormat(t *testing.T) {
	ev := &Event{Format: "elimination"}
	if got := ev.GetDivisionFormat("d1"); got != "elimination" {
		t.Errorf("Expected fallback elimination, got %s", got)
	}

	ev.DivisionConfigs = map[string]DivisionConfig{
		"d1": {Format: "round_robin"},
		"d2": {Format: ""},
	}
	if got := ev.GetDivisionFormat("d1"); got != "round_robin" {
		t.Errorf("Expected division override round_robin, got %s", got)
	}
	if got := ev.GetDivisionFormat("d2"); got != "elimination" {
		t.Errorf("Expected fallback when division format empty, got %s", got)
	}
	if got := ev.GetDivisionFormat("missing"); got != "elimination" {
		t.Errorf("Expected fallback for missing division, got %s", got)
	}
}

func TestGetGroupPassCount(t *testing.T) {
	ev := &Event{GroupPassCount: 2}
	if got := ev.GetGroupPassCount("d1"); got != 2 {
		t.Errorf("Expected fallback 2, got %d", got)
	}

	ev.DivisionConfigs = map[string]DivisionConfig{
		"d1": {GroupPassCount: 4},
		"d2": {GroupPassCount: 0},
	}
	if got := ev.GetGroupPassCount("d1"); got != 4 {
		t.Errorf("Expected division override 4, got %d", got)
	}
	if got := ev.GetGroupPassCount("d2"); got != 2 {
		t.Errorf("Expected fallback to global 2 when division value is 0, got %d", got)
	}
	if got := ev.GetGroupPassCount("missing"); got != 2 {
		t.Errorf("Expected fallback 2 for missing division, got %d", got)
	}
}

func TestGetLosersGroupPassCount(t *testing.T) {
	ev := &Event{LosersGroupPassCount: 1}
	if got := ev.GetLosersGroupPassCount("d1"); got != 1 {
		t.Errorf("Expected fallback 1, got %d", got)
	}

	ev.DivisionConfigs = map[string]DivisionConfig{
		"d1": {LosersGroupPassCount: 3},
	}
	if got := ev.GetLosersGroupPassCount("d1"); got != 3 {
		t.Errorf("Expected division override 3, got %d", got)
	}
	if got := ev.GetLosersGroupPassCount("missing"); got != 1 {
		t.Errorf("Expected fallback 1 for missing division, got %d", got)
	}
}

func TestGetGroupCount(t *testing.T) {
	ev := &Event{}
	if got := ev.GetGroupCount("d1"); got != 0 {
		t.Errorf("Expected 0 with no config, got %d", got)
	}

	ev.DivisionConfigs = map[string]DivisionConfig{
		"d1": {GroupCount: 5},
		"d2": {GroupCount: 0},
	}
	if got := ev.GetGroupCount("d1"); got != 5 {
		t.Errorf("Expected 5, got %d", got)
	}
	if got := ev.GetGroupCount("d2"); got != 0 {
		t.Errorf("Expected 0 when division value is 0, got %d", got)
	}
}

func TestAddFindRemoveMatch(t *testing.T) {
	ev := &Event{}
	ev.AddMatch(Match{ID: "m1"})
	ev.AddMatch(Match{ID: "m2"})

	if len(ev.Matches) != 2 {
		t.Fatalf("Expected 2 matches, got %d", len(ev.Matches))
	}

	m, err := ev.FindMatch("m1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if m.ID != "m1" {
		t.Errorf("Expected m1, got %s", m.ID)
	}

	_, err = ev.FindMatch("missing")
	if err == nil {
		t.Errorf("Expected error for missing match")
	}

	if err := ev.RemoveMatch("m1"); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(ev.Matches) != 1 {
		t.Errorf("Expected 1 match remaining, got %d", len(ev.Matches))
	}

	if err := ev.RemoveMatch("missing"); err == nil {
		t.Errorf("Expected error removing missing match")
	}
}

func TestHasMatchesStarted(t *testing.T) {
	ev := &Event{}
	if ev.HasMatchesStarted() {
		t.Errorf("Expected false for no matches")
	}

	ev.Matches = []Match{{ID: "m1", Status: "scheduled"}}
	if ev.HasMatchesStarted() {
		t.Errorf("Expected false for scheduled matches")
	}

	ev.Matches = append(ev.Matches, Match{ID: "m2", Status: "in_progress"})
	if !ev.HasMatchesStarted() {
		t.Errorf("Expected true when a match is in_progress")
	}

	ev.Matches = []Match{{ID: "m3", Status: "finished"}}
	if !ev.HasMatchesStarted() {
		t.Errorf("Expected true when a match is finished")
	}
}

func TestMovePlayer_Locked(t *testing.T) {
	ev := &Event{ManualSeedingLocked: true}
	if err := ev.MovePlayer("p1", "g1", 0); err == nil {
		t.Errorf("Expected error when seeding locked")
	}
}

func TestMovePlayer_PlayerNotRegistered(t *testing.T) {
	ev := &Event{Type: "singles"}
	if err := ev.MovePlayer("missing", "g1", 0); err == nil {
		t.Errorf("Expected error for unregistered player")
	}
}

func TestMovePlayer_TeamNotRegistered(t *testing.T) {
	ev := &Event{Type: "teams"}
	if err := ev.MovePlayer("missing-team", "g1", 0); err == nil {
		t.Errorf("Expected error for unregistered team")
	}
}

func TestMovePlayer_TargetGroupNotFound(t *testing.T) {
	p1 := &player.Player{ID: "p1"}
	ev := &Event{
		Type:         "singles",
		Participants: []*player.Player{p1},
		Groups: []Group{
			{ID: "g1", Players: []*player.Player{p1}},
		},
	}
	if err := ev.MovePlayer("p1", "missing-group", 0); err == nil {
		t.Errorf("Expected error for missing target group")
	}
}

func TestMovePlayer_AlreadyInTargetGroup(t *testing.T) {
	// Simulates a data inconsistency where the player is listed in two groups:
	// the source-removal pass strips the first occurrence (g1), leaving the
	// duplicate entry in the target group (g2) to trigger the "already in
	// target" guard.
	p1 := &player.Player{ID: "p1"}
	ev := &Event{
		Type:         "singles",
		Participants: []*player.Player{p1},
		Groups: []Group{
			{ID: "g1", Players: []*player.Player{p1}},
			{ID: "g2", Players: []*player.Player{p1}},
		},
	}
	if err := ev.MovePlayer("p1", "g2", 0); err == nil {
		t.Errorf("Expected error, player already in target group")
	}
}

func TestMovePlayer_MovesBetweenGroups(t *testing.T) {
	p1 := &player.Player{ID: "p1"}
	p2 := &player.Player{ID: "p2"}
	ev := &Event{
		Type:         "singles",
		Participants: []*player.Player{p1, p2},
		Groups: []Group{
			{ID: "g1", Players: []*player.Player{p1}},
			{ID: "g2", Players: []*player.Player{p2}},
		},
	}
	if err := ev.MovePlayer("p1", "g2", 0); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(ev.Groups[0].Players) != 0 {
		t.Errorf("Expected g1 to be empty, got %d players", len(ev.Groups[0].Players))
	}
	if len(ev.Groups[1].Players) != 2 {
		t.Fatalf("Expected g2 to have 2 players, got %d", len(ev.Groups[1].Players))
	}
	if ev.Groups[1].Players[0].ID != "p1" {
		t.Errorf("Expected p1 inserted at index 0, got %s", ev.Groups[1].Players[0].ID)
	}
}

func TestMovePlayer_EmptyTargetDefaultsToSource(t *testing.T) {
	p1 := &player.Player{ID: "p1"}
	p2 := &player.Player{ID: "p2"}
	ev := &Event{
		Type:         "singles",
		Participants: []*player.Player{p1, p2},
		Groups: []Group{
			{ID: "g1", Players: []*player.Player{p1, p2}},
		},
	}
	// Moving within the same group by re-inserting at a different index:
	// first removed from source (g1), then since target is empty it defaults
	// back to the source group id, and since p1 was already removed it succeeds.
	if err := ev.MovePlayer("p1", "", 1); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(ev.Groups[0].Players) != 2 {
		t.Fatalf("Expected 2 players still in g1, got %d", len(ev.Groups[0].Players))
	}
	if ev.Groups[0].Players[1].ID != "p1" {
		t.Errorf("Expected p1 to be reinserted at index 1, got %s", ev.Groups[0].Players[1].ID)
	}
}

func TestMovePlayer_TeamMove(t *testing.T) {
	team := &Team{ID: "team1", Name: "Team One", Players: []*player.Player{{SinglesElo: 1200, DoublesElo: 1100}}}
	ev := &Event{
		Type:  "teams",
		Teams: []*Team{team},
		Groups: []Group{
			{ID: "g1"},
			{ID: "g2"},
		},
	}
	if err := ev.MovePlayer("team1", "g1", 0); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(ev.Groups[0].Players) != 1 {
		t.Fatalf("Expected 1 team-player in g1, got %d", len(ev.Groups[0].Players))
	}
	if ev.Groups[0].Players[0].ID != "team1" {
		t.Errorf("Expected inserted player ID team1, got %s", ev.Groups[0].Players[0].ID)
	}
}

func TestMovePlayer_NegativeOrOutOfRangeIndex(t *testing.T) {
	p1 := &player.Player{ID: "p1"}
	p2 := &player.Player{ID: "p2"}
	ev := &Event{
		Type:         "singles",
		Participants: []*player.Player{p1, p2},
		Groups: []Group{
			{ID: "g1", Players: []*player.Player{p1}},
			{ID: "g2", Players: []*player.Player{}},
		},
	}
	if err := ev.MovePlayer("p1", "g2", -5); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(ev.Groups[1].Players) != 1 {
		t.Fatalf("Expected 1 player in g2, got %d", len(ev.Groups[1].Players))
	}
}
