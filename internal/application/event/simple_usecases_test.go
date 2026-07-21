package event

import (
	"context"
	"errors"
	"testing"

	tournamentDomain "table-tennis-backend/internal/domain/event"
)

func TestDeleteTournamentUseCase_Execute(t *testing.T) {
	repo := newMockRepo()
	repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
	uc := NewDeleteTournamentUseCase(repo)

	if err := uc.Execute(context.Background(), "t1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.deleteCalls != 1 {
		t.Errorf("expected 1 delete call, got %d", repo.deleteCalls)
	}
	if _, ok := repo.events["t1"]; ok {
		t.Errorf("expected event to be deleted")
	}
}

func TestDeleteTournamentUseCase_Execute_Error(t *testing.T) {
	repo := newMockRepo()
	repo.deleteErr = errors.New("db error")
	uc := NewDeleteTournamentUseCase(repo)

	if err := uc.Execute(context.Background(), "t1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRemoveParticipantUseCase_Execute(t *testing.T) {
	repo := newMockRepo()
	uc := NewRemoveParticipantUseCase(repo)

	if err := uc.Execute(context.Background(), "t1", "p1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	repo.removeParticiErr = errors.New("boom")
	if err := uc.Execute(context.Background(), "t1", "p1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetTournamentsUseCase_Execute(t *testing.T) {
	repo := newMockRepo()
	repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
	repo.events["t2"] = &tournamentDomain.Event{ID: "t2"}
	uc := NewGetTournamentsUseCase(repo)

	res, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 events, got %d", len(res))
	}
}

func TestGetTournamentsUseCase_Execute_Error(t *testing.T) {
	repo := newMockRepo()
	repo.getAllErr = errors.New("db down")
	uc := NewGetTournamentsUseCase(repo)

	if _, err := uc.Execute(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetOccupiedTablesUseCase_Execute(t *testing.T) {
	matchRepo := &mockMatchRepo{occupiedByEvent: []int{1, 2}, occupiedByTourn: []int{3}}
	uc := NewGetOccupiedTablesUseCase(matchRepo)

	t.Run("nil event returns nil, no error", func(t *testing.T) {
		res, err := uc.Execute(context.Background(), nil)
		if err != nil || res != nil {
			t.Fatalf("expected nil, nil, got %v, %v", res, err)
		}
	})

	t.Run("event with EventID uses GetOccupiedTablesByEvent", func(t *testing.T) {
		eid := "e1"
		res, err := uc.Execute(context.Background(), &tournamentDomain.Event{ID: "t1", EventID: &eid})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(res) != 2 {
			t.Fatalf("expected 2 tables, got %v", res)
		}
	})

	t.Run("event without EventID uses GetOccupiedTablesByTournament", func(t *testing.T) {
		res, err := uc.Execute(context.Background(), &tournamentDomain.Event{ID: "t1"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(res) != 1 {
			t.Fatalf("expected 1 table, got %v", res)
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		errRepo := &mockMatchRepo{occupiedByTournErr: errors.New("boom")}
		uc := NewGetOccupiedTablesUseCase(errRepo)
		_, err := uc.Execute(context.Background(), &tournamentDomain.Event{ID: "t1"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestToggleSeedingLockUseCase_Execute(t *testing.T) {
	repo := newMockRepo()
	repo.events["t1"] = &tournamentDomain.Event{ID: "t1", ManualSeedingLocked: false}
	uc := NewToggleSeedingLockUseCase(repo)

	if err := uc.Execute(context.Background(), "t1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.events["t1"].ManualSeedingLocked {
		t.Errorf("expected lock to be toggled to true")
	}

	if err := uc.Execute(context.Background(), "t1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.events["t1"].ManualSeedingLocked {
		t.Errorf("expected lock to be toggled back to false")
	}
}

func TestToggleSeedingLockUseCase_Execute_GetError(t *testing.T) {
	repo := newMockRepo()
	repo.getErr = errors.New("not found")
	uc := NewToggleSeedingLockUseCase(repo)

	if err := uc.Execute(context.Background(), "missing"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestToggleSeedingLockUseCase_Execute_UpdateError(t *testing.T) {
	repo := newMockRepo()
	repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
	repo.updateErr = errors.New("db error")
	uc := NewToggleSeedingLockUseCase(repo)

	if err := uc.Execute(context.Background(), "t1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTeamUseCases(t *testing.T) {
	t.Run("create team success", func(t *testing.T) {
		repo := newMockRepo()
		uc := NewCreateTeamUseCase(repo)
		team, err := uc.Execute(context.Background(), "t1", "Team A")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if team.Name != "Team A" {
			t.Errorf("expected Team A, got %s", team.Name)
		}
	})

	t.Run("create team empty name error", func(t *testing.T) {
		repo := newMockRepo()
		uc := NewCreateTeamUseCase(repo)
		_, err := uc.Execute(context.Background(), "t1", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("create team save error", func(t *testing.T) {
		repo := newMockRepo()
		repo.saveTeamErr = errors.New("db error")
		uc := NewCreateTeamUseCase(repo)
		_, err := uc.Execute(context.Background(), "t1", "Team A")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("delete team", func(t *testing.T) {
		repo := newMockRepo()
		uc := NewDeleteTeamUseCase(repo)
		if err := uc.Execute(context.Background(), "team1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		repo.deleteTeamErr = errors.New("boom")
		if err := uc.Execute(context.Background(), "team1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("assign player to team", func(t *testing.T) {
		repo := newMockRepo()
		uc := NewAssignPlayerToTeamUseCase(repo)
		if err := uc.Execute(context.Background(), "team1", "p1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		repo.addToTeamErr = errors.New("boom")
		if err := uc.Execute(context.Background(), "team1", "p1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("remove player from team", func(t *testing.T) {
		repo := newMockRepo()
		uc := NewRemovePlayerFromTeamUseCase(repo)
		if err := uc.Execute(context.Background(), "team1", "p1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		repo.removeFromTeamErr = errors.New("boom")
		if err := uc.Execute(context.Background(), "team1", "p1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
