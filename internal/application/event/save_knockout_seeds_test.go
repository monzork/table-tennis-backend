package event

import (
	"context"
	"errors"
	"testing"

	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func TestSaveKnockoutSeedsUseCase_Execute(t *testing.T) {
	newUC := func() (*SaveKnockoutSeedsUseCase, *mockRepo, *mockDivisionRepo) {
		repo := newMockRepo()
		divRepo := &mockDivisionRepo{}
		uc := NewSaveKnockoutSeedsUseCase(repo, divRepo)
		return uc, repo, divRepo
	}

	t.Run("invalid JSON returns error", func(t *testing.T) {
		uc, _, _ := newUC()
		err := uc.Execute(context.Background(), "t1", "d1", "not-json")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("get event error propagates", func(t *testing.T) {
		uc, repo, _ := newUC()
		repo.getErr = errors.New("db error")
		err := uc.Execute(context.Background(), "t1", "d1", `["p1"]`)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("division get all error propagates", func(t *testing.T) {
		uc, repo, divRepo := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		divRepo.getAllErr = errors.New("boom")
		err := uc.Execute(context.Background(), "t1", "d1", `["p1"]`)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("creates new knockout group with resolved division name", func(t *testing.T) {
		uc, repo, divRepo := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A"}
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Participants: []*playerDomain.Player{p1}}
		divRepo.divisions = []*divisionDomain.Division{{ID: "d1", Name: "First Division"}}

		if err := uc.Execute(context.Background(), "t1", "d1", `["p1"]`); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// "First Division" has its " Division" suffix stripped, so the group is "First - Knockout Seeds".
		found := false
		for _, g := range repo.events["t1"].Groups {
			if g.Name == "First - Knockout Seeds" {
				found = true
				if len(g.Players) != 1 || g.Players[0].ID != "p1" {
					t.Errorf("expected p1 in knockout group, got %+v", g.Players)
				}
			}
		}
		if !found {
			t.Errorf("expected knockout group to be created, got groups %+v", repo.events["t1"].Groups)
		}
	})

	t.Run("empty divID falls back to Open Bracket when SkipElo", func(t *testing.T) {
		uc, repo, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A"}
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", SkipElo: true, Participants: []*playerDomain.Player{p1}}

		if err := uc.Execute(context.Background(), "t1", "", `["p1"]`); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		found := false
		for _, g := range repo.events["t1"].Groups {
			if g.Name == "Open Bracket - Knockout Seeds" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Open Bracket knockout group")
		}
	})

	t.Run("empty divID falls back to Unclassified when not SkipElo", func(t *testing.T) {
		uc, repo, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A"}
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Participants: []*playerDomain.Player{p1}}

		if err := uc.Execute(context.Background(), "t1", "", `["p1"]`); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		found := false
		for _, g := range repo.events["t1"].Groups {
			if g.Name == "Unclassified - Knockout Seeds" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Unclassified knockout group")
		}
	})

	t.Run("resolves team players when player not found in participants", func(t *testing.T) {
		uc, repo, _ := newUC()
		team := &tournamentDomain.Team{ID: "team1", Name: "Team X"}
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", SkipElo: true, Teams: []*tournamentDomain.Team{team}}

		if err := uc.Execute(context.Background(), "t1", "", `["team1"]`); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		var got *tournamentDomain.Group
		for i := range repo.events["t1"].Groups {
			if repo.events["t1"].Groups[i].Name == "Open Bracket - Knockout Seeds" {
				got = &repo.events["t1"].Groups[i]
			}
		}
		if got == nil || len(got.Players) != 1 || got.Players[0].ID != "team1" {
			t.Fatalf("expected team1 resolved as player, got %+v", got)
		}
	})

	t.Run("updates existing knockout group in place", func(t *testing.T) {
		uc, repo, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A"}
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", SkipElo: true, Participants: []*playerDomain.Player{p1},
			Groups: []tournamentDomain.Group{{ID: "g1", Name: "Open Bracket - Knockout Seeds", Players: nil}},
		}

		if err := uc.Execute(context.Background(), "t1", "", `["p1"]`); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(repo.events["t1"].Groups) != 1 {
			t.Fatalf("expected group count to stay 1 (updated in place), got %d", len(repo.events["t1"].Groups))
		}
		if len(repo.events["t1"].Groups[0].Players) != 1 {
			t.Errorf("expected 1 player in updated group")
		}
	})

	t.Run("update groups error propagates", func(t *testing.T) {
		uc, repo, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A"}
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", SkipElo: true, Participants: []*playerDomain.Player{p1}}
		repo.updateGroupsErr = errors.New("boom")

		if err := uc.Execute(context.Background(), "t1", "", `["p1"]`); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
