package tournament

import (
	"context"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

type CreateTeamUseCase struct {
	repo *bun.TournamentRepository
}

func NewCreateTeamUseCase(repo *bun.TournamentRepository) *CreateTeamUseCase {
	return &CreateTeamUseCase{repo: repo}
}

func (uc *CreateTeamUseCase) Execute(ctx context.Context, tournamentIDStr string, name string) (*tournamentDomain.Team, error) {
	tourneyID, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		return nil, err
	}

	team, err := tournamentDomain.NewTeam(tourneyID, name)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.SaveTeam(ctx, team); err != nil {
		return nil, err
	}

	return team, nil
}

type DeleteTeamUseCase struct {
	repo *bun.TournamentRepository
}

func NewDeleteTeamUseCase(repo *bun.TournamentRepository) *DeleteTeamUseCase {
	return &DeleteTeamUseCase{repo: repo}
}

func (uc *DeleteTeamUseCase) Execute(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	return uc.repo.DeleteTeam(ctx, id)
}

type AssignPlayerToTeamUseCase struct {
	repo *bun.TournamentRepository
}

func NewAssignPlayerToTeamUseCase(repo *bun.TournamentRepository) *AssignPlayerToTeamUseCase {
	return &AssignPlayerToTeamUseCase{repo: repo}
}

func (uc *AssignPlayerToTeamUseCase) Execute(ctx context.Context, teamIDStr string, playerIDStr string) error {
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		return err
	}
	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		return err
	}
	return uc.repo.AddPlayerToTeam(ctx, teamID, playerID)
}

type RemovePlayerFromTeamUseCase struct {
	repo *bun.TournamentRepository
}

func NewRemovePlayerFromTeamUseCase(repo *bun.TournamentRepository) *RemovePlayerFromTeamUseCase {
	return &RemovePlayerFromTeamUseCase{repo: repo}
}

func (uc *RemovePlayerFromTeamUseCase) Execute(ctx context.Context, teamIDStr string, playerIDStr string) error {
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		return err
	}
	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		return err
	}
	return uc.repo.RemovePlayerFromTeam(ctx, teamID, playerID)
}
