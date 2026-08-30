package event

import (
	"table-tennis-backend/internal/domain/player"
	"testing"
)

func TestPlacementEloBonus_Elimination(t *testing.T) {
	john := &player.Player{ID: "john"}
	jane := &player.Player{ID: "jane"}
	bob := &player.Player{ID: "bob"}
	alice := &player.Player{ID: "alice"}
	charlie := &player.Player{ID: "charlie"}

	ev := &Event{
		Format: "elimination",
		Type:   "singles",
		Matches: []Match{
			{
				Stage:      "final",
				Status:     "finished",
				WinnerTeam: "A",
				TeamA:      []*player.Player{john},
				TeamB:      []*player.Player{jane},
			},
			{
				Stage:      "semifinal",
				Status:     "finished",
				WinnerTeam: "A",
				TeamA:      []*player.Player{bob},
				TeamB:      []*player.Player{alice}, // semi loser -> 3rd
			},
			{
				Stage:      "semifinal",
				Status:     "finished",
				WinnerTeam: "B",
				TeamA:      []*player.Player{charlie}, // semi loser -> 3rd
				TeamB:      []*player.Player{john},
			},
		},
	}

	bonus := PlacementEloBonus(ev)
	if bonus["john"] != FirstPlaceEloBonus {
		t.Errorf("expected champion bonus %.1f, got %v", FirstPlaceEloBonus, bonus["john"])
	}
	if bonus["jane"] != SecondPlaceEloBonus {
		t.Errorf("expected runner-up bonus %.1f, got %v", SecondPlaceEloBonus, bonus["jane"])
	}
	if bonus["alice"] != ThirdPlaceEloBonus || bonus["charlie"] != ThirdPlaceEloBonus {
		t.Errorf("expected both semifinal losers at %.1f, got alice=%v charlie=%v", ThirdPlaceEloBonus, bonus["alice"], bonus["charlie"])
	}
	if _, ok := bonus["bob"]; ok {
		t.Errorf("expected bob (a semifinal winner, not on the podium) to have no bonus entry, got %v", bonus["bob"])
	}
}

func TestPlacementEloBonus_NoFinishedFinal(t *testing.T) {
	ev := &Event{
		Format: "elimination",
		Type:   "singles",
		Matches: []Match{
			{Stage: "final", Status: "in_progress", TeamA: []*player.Player{{ID: "a"}}, TeamB: []*player.Player{{ID: "b"}}},
		},
	}
	bonus := PlacementEloBonus(ev)
	if len(bonus) != 0 {
		t.Errorf("expected no bonus for unfinished final, got %v", bonus)
	}
}

func TestPlacementEloBonus_RoundRobin(t *testing.T) {
	p1 := &player.Player{ID: "p1"}
	p2 := &player.Player{ID: "p2"}
	p3 := &player.Player{ID: "p3"}
	p4 := &player.Player{ID: "p4"}

	ev := &Event{
		Format:       "round_robin",
		Type:         "singles",
		Participants: []*player.Player{p1, p2, p3, p4},
		Matches: []Match{
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p3}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p4}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p3}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p4}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p3}, TeamB: []*player.Player{p4}},
		},
	}

	bonus := PlacementEloBonus(ev)
	if bonus["p1"] != FirstPlaceEloBonus {
		t.Errorf("expected p1 (3 wins) as champion bonus %.1f, got %v", FirstPlaceEloBonus, bonus["p1"])
	}
	if bonus["p2"] != SecondPlaceEloBonus {
		t.Errorf("expected p2 (2 wins) as runner-up bonus %.1f, got %v", SecondPlaceEloBonus, bonus["p2"])
	}
	if bonus["p3"] != ThirdPlaceEloBonus {
		t.Errorf("expected p3 (1 win) at bonus %.1f, got %v", ThirdPlaceEloBonus, bonus["p3"])
	}
	if _, ok := bonus["p4"]; ok {
		t.Errorf("expected last place (p4) to have no bonus entry, got %v", bonus["p4"])
	}
}

func TestPlacementEloBonus_TeamsFormat(t *testing.T) {
	johnM := &player.Player{ID: "john"}
	janeM := &player.Player{ID: "jane"}
	bobM := &player.Player{ID: "bob"}
	aliceM := &player.Player{ID: "alice"}

	teamA := &Team{ID: "teamA", Name: "Team A", Players: []*player.Player{johnM}}
	teamB := &Team{ID: "teamB", Name: "Team B", Players: []*player.Player{janeM}}
	teamC := &Team{ID: "teamC", Name: "Team C", Players: []*player.Player{bobM}}
	teamD := &Team{ID: "teamD", Name: "Team D", Players: []*player.Player{aliceM}}

	ev := &Event{
		Format: "elimination",
		Type:   "teams",
		Teams:  []*Team{teamA, teamB, teamC, teamD},
		Matches: []Match{
			{
				Stage:      "final",
				Status:     "finished",
				MatchType:  "teams",
				WinnerTeam: "A",
				TeamA:      []*player.Player{{ID: "teamA", FirstName: "Team A"}},
				TeamB:      []*player.Player{{ID: "teamB", FirstName: "Team B"}},
			},
			{
				Stage:      "semifinal",
				Status:     "finished",
				MatchType:  "teams",
				WinnerTeam: "A",
				TeamA:      []*player.Player{{ID: "teamC", FirstName: "Team C"}},
				TeamB:      []*player.Player{{ID: "teamD", FirstName: "Team D"}},
			},
		},
	}

	bonus := PlacementEloBonus(ev)
	if bonus["john"] != FirstPlaceEloBonus {
		t.Errorf("expected team A member john as champion bonus %.1f, got %v", FirstPlaceEloBonus, bonus["john"])
	}
	if bonus["jane"] != SecondPlaceEloBonus {
		t.Errorf("expected team B member jane as runner-up bonus %.1f, got %v", SecondPlaceEloBonus, bonus["jane"])
	}
	if bonus["alice"] != ThirdPlaceEloBonus {
		t.Errorf("expected team D member alice (eliminated in semis) at bonus %.1f, got %v", ThirdPlaceEloBonus, bonus["alice"])
	}
}

func TestPlacementEloBonus_UnknownFormat(t *testing.T) {
	ev := &Event{Format: "single_division_multiple_brackets"}
	if bonus := PlacementEloBonus(ev); bonus != nil {
		t.Errorf("expected nil bonus for unhandled format, got %v", bonus)
	}
}

func TestPlacementEloBonus_RoundRobinEmptyParticipants(t *testing.T) {
	ev := &Event{Format: "round_robin", Type: "singles"}
	if bonus := PlacementEloBonus(ev); bonus != nil {
		t.Errorf("expected nil bonus with no participants, got %v", bonus)
	}
}
