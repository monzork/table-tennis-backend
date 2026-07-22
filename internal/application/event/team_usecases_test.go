package event_test

import (
	"context"
	"errors"
	"testing"

	appevent "table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/domain/event"
)

// mockTeamRepo implements only event.TeamRepository (4 methods) rather than the full
// 20-method event.Repository — the concrete payoff of splitting the "god" repository
// interface: these use cases can now be tested without stubbing unrelated capability.
type mockTeamRepo struct {
	savedTeam       *event.Team
	deletedID       string
	addedTeamID     string
	addedPlayerID   string
	removedTeamID   string
	removedPlayerID string
	err             error
}

func (m *mockTeamRepo) SaveTeam(ctx context.Context, team *event.Team) error {
	if m.err != nil {
		return m.err
	}
	m.savedTeam = team
	return nil
}

func (m *mockTeamRepo) DeleteTeam(ctx context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	m.deletedID = id
	return nil
}

func (m *mockTeamRepo) AddPlayerToTeam(ctx context.Context, teamID string, playerID string) error {
	if m.err != nil {
		return m.err
	}
	m.addedTeamID, m.addedPlayerID = teamID, playerID
	return nil
}

func (m *mockTeamRepo) RemovePlayerFromTeam(ctx context.Context, teamID string, playerID string) error {
	if m.err != nil {
		return m.err
	}
	m.removedTeamID, m.removedPlayerID = teamID, playerID
	return nil
}

func TestCreateTeamUseCase(t *testing.T) {
	repo := &mockTeamRepo{}
	uc := appevent.NewCreateTeamUseCase(repo)

	team, err := uc.Execute(context.Background(), "tournament-1", "Squad A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if team.Name != "Squad A" || team.TournamentID != "tournament-1" {
		t.Errorf("unexpected team: %+v", team)
	}
	if repo.savedTeam == nil || repo.savedTeam.ID != team.ID {
		t.Errorf("expected team to be saved via repo, got %+v", repo.savedTeam)
	}
}

func TestCreateTeamUseCase_EmptyName(t *testing.T) {
	repo := &mockTeamRepo{}
	uc := appevent.NewCreateTeamUseCase(repo)

	if _, err := uc.Execute(context.Background(), "tournament-1", ""); err == nil {
		t.Fatal("expected error for empty team name")
	}
	if repo.savedTeam != nil {
		t.Error("expected no save attempt when name validation fails")
	}
}

func TestDeleteTeamUseCase(t *testing.T) {
	repo := &mockTeamRepo{}
	uc := appevent.NewDeleteTeamUseCase(repo)

	if err := uc.Execute(context.Background(), "team-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.deletedID != "team-1" {
		t.Errorf("expected team-1 to be deleted, got %q", repo.deletedID)
	}
}

func TestAssignPlayerToTeamUseCase(t *testing.T) {
	repo := &mockTeamRepo{}
	uc := appevent.NewAssignPlayerToTeamUseCase(repo)

	if err := uc.Execute(context.Background(), "team-1", "player-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.addedTeamID != "team-1" || repo.addedPlayerID != "player-1" {
		t.Errorf("expected player-1 added to team-1, got team=%q player=%q", repo.addedTeamID, repo.addedPlayerID)
	}
}

func TestRemovePlayerFromTeamUseCase(t *testing.T) {
	repo := &mockTeamRepo{}
	uc := appevent.NewRemovePlayerFromTeamUseCase(repo)

	if err := uc.Execute(context.Background(), "team-1", "player-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.removedTeamID != "team-1" || repo.removedPlayerID != "player-1" {
		t.Errorf("expected player-1 removed from team-1, got team=%q player=%q", repo.removedTeamID, repo.removedPlayerID)
	}
}

func TestTeamUseCases_RepositoryError(t *testing.T) {
	repo := &mockTeamRepo{err: errors.New("db down")}

	if _, err := appevent.NewCreateTeamUseCase(repo).Execute(context.Background(), "t1", "Squad A"); err == nil {
		t.Error("expected CreateTeamUseCase to propagate repository error")
	}
	if err := appevent.NewDeleteTeamUseCase(repo).Execute(context.Background(), "team-1"); err == nil {
		t.Error("expected DeleteTeamUseCase to propagate repository error")
	}
	if err := appevent.NewAssignPlayerToTeamUseCase(repo).Execute(context.Background(), "t", "p"); err == nil {
		t.Error("expected AssignPlayerToTeamUseCase to propagate repository error")
	}
	if err := appevent.NewRemovePlayerFromTeamUseCase(repo).Execute(context.Background(), "t", "p"); err == nil {
		t.Error("expected RemovePlayerFromTeamUseCase to propagate repository error")
	}
}
