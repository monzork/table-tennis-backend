package event

import (
	"context"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
)

type CreateTeamUseCase struct {
	repo tournamentDomain.TeamRepository
}

func NewCreateTeamUseCase(repo tournamentDomain.TeamRepository) *CreateTeamUseCase {
	return &CreateTeamUseCase{repo: repo}
}

func (uc *CreateTeamUseCase) Execute(ctx context.Context, tournamentIDStr string, name string) (*tournamentDomain.Team, error) {
	team, err := tournamentDomain.NewTeam(idgen.Generate(), tournamentIDStr, name)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.SaveTeam(ctx, team); err != nil {
		return nil, err
	}

	return team, nil
}

type DeleteTeamUseCase struct {
	repo tournamentDomain.TeamRepository
}

func NewDeleteTeamUseCase(repo tournamentDomain.TeamRepository) *DeleteTeamUseCase {
	return &DeleteTeamUseCase{repo: repo}
}

func (uc *DeleteTeamUseCase) Execute(ctx context.Context, idStr string) error {
	return uc.repo.DeleteTeam(ctx, idStr)
}

type AssignPlayerToTeamUseCase struct {
	repo tournamentDomain.TeamRepository
}

func NewAssignPlayerToTeamUseCase(repo tournamentDomain.TeamRepository) *AssignPlayerToTeamUseCase {
	return &AssignPlayerToTeamUseCase{repo: repo}
}

func (uc *AssignPlayerToTeamUseCase) Execute(ctx context.Context, teamIDStr string, playerIDStr string) error {
	return uc.repo.AddPlayerToTeam(ctx, teamIDStr, playerIDStr)
}

type RemovePlayerFromTeamUseCase struct {
	repo tournamentDomain.TeamRepository
}

func NewRemovePlayerFromTeamUseCase(repo tournamentDomain.TeamRepository) *RemovePlayerFromTeamUseCase {
	return &RemovePlayerFromTeamUseCase{repo: repo}
}

func (uc *RemovePlayerFromTeamUseCase) Execute(ctx context.Context, teamIDStr string, playerIDStr string) error {
	return uc.repo.RemovePlayerFromTeam(ctx, teamIDStr, playerIDStr)
}
