package event

import (
	"context"
	"errors"
	"testing"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func TestStartKnockoutStageUseCase_Execute(t *testing.T) {
	newUC := func() (*StartKnockoutStageUseCase, *mockRepo, *mockMatchRepo, *mockDivisionRepo) {
		repo := newMockRepo()
		matchRepo := &mockMatchRepo{}
		divRepo := &mockDivisionRepo{}
		uc := NewStartKnockoutStageUseCase(repo, matchRepo, divRepo)
		return uc, repo, matchRepo, divRepo
	}

	t.Run("get error propagates", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		repo.getErr = errors.New("db error")
		if err := uc.Execute(context.Background(), "t1", "d1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("finished event rejects start", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Status: "finished"}
		if err := uc.Execute(context.Background(), "t1", "d1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("division repo error propagates", func(t *testing.T) {
		uc, repo, _, divRepo := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		divRepo.getAllErr = errors.New("boom")
		if err := uc.Execute(context.Background(), "t1", "d1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("no matches to start when group stage not finished", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A"}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "B", LastName: "B"}
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", Format: "groups_elimination", SkipElo: true,
			Participants: []*playerDomain.Player{p1, p2},
			Groups:       []tournamentDomain.Group{{ID: "g1", Name: "Group A", Players: []*playerDomain.Player{p1, p2}}},
		}
		if err := uc.Execute(context.Background(), "t1", ""); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("happy path schedules first round matches for finished group", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", SinglesElo: 1200}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", SinglesElo: 1100}
		m := tournamentDomain.Match{
			ID: "gm1", MatchType: "singles", Stage: "group",
			TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2},
			Status: "finished", WinnerTeam: "A",
		}
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", Format: "groups_elimination", Type: "singles", SkipElo: true, GroupPassCount: 2,
			Participants: []*playerDomain.Player{p1, p2},
			Groups:       []tournamentDomain.Group{{ID: "g1", Name: "Group A", Players: []*playerDomain.Player{p1, p2}}},
			Matches:      []tournamentDomain.Match{m},
		}

		if err := uc.Execute(context.Background(), "t1", ""); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(matchRepo.savedMatches) == 0 {
			t.Fatalf("expected first round match(es) to be saved")
		}
	})

	t.Run("save error propagates on happy path", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", SinglesElo: 1200}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", SinglesElo: 1100}
		m := tournamentDomain.Match{
			ID: "gm1", MatchType: "singles", Stage: "group",
			TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2},
			Status: "finished", WinnerTeam: "A",
		}
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", Format: "groups_elimination", Type: "singles", SkipElo: true, GroupPassCount: 2,
			Participants: []*playerDomain.Player{p1, p2},
			Groups:       []tournamentDomain.Group{{ID: "g1", Name: "Group A", Players: []*playerDomain.Player{p1, p2}}},
			Matches:      []tournamentDomain.Match{m},
		}
		matchRepo.saveErr = errors.New("boom")

		if err := uc.Execute(context.Background(), "t1", ""); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSameTeams(t *testing.T) {
	p1 := &playerDomain.Player{ID: "p1"}
	p2 := &playerDomain.Player{ID: "p2"}
	p3 := &playerDomain.Player{ID: "p3"}

	m1 := tournamentDomain.Match{TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}}

	t.Run("identical teams match", func(t *testing.T) {
		m2 := tournamentDomain.Match{TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}}
		if !sameTeams(m1, m2) {
			t.Errorf("expected teams to match")
		}
	})

	t.Run("different team sizes do not match", func(t *testing.T) {
		m2 := tournamentDomain.Match{TeamA: []*playerDomain.Player{p1, p3}, TeamB: []*playerDomain.Player{p2}}
		if sameTeams(m1, m2) {
			t.Errorf("expected size mismatch to not match")
		}
	})

	t.Run("different team A player does not match", func(t *testing.T) {
		m2 := tournamentDomain.Match{TeamA: []*playerDomain.Player{p3}, TeamB: []*playerDomain.Player{p2}}
		if sameTeams(m1, m2) {
			t.Errorf("expected different TeamA to not match")
		}
	})

	t.Run("different team B player does not match", func(t *testing.T) {
		m2 := tournamentDomain.Match{TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p3}}
		if sameTeams(m1, m2) {
			t.Errorf("expected different TeamB to not match")
		}
	})

	t.Run("empty teams match", func(t *testing.T) {
		empty1 := tournamentDomain.Match{}
		empty2 := tournamentDomain.Match{}
		if !sameTeams(empty1, empty2) {
			t.Errorf("expected empty teams to match")
		}
	})
}
