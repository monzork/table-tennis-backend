package tournament

import (
	"context"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

type GetTournamentsUseCase struct {
	repo tournamentDomain.Repository
}

func NewGetTournamentsUseCase(repo tournamentDomain.Repository) *GetTournamentsUseCase {
	return &GetTournamentsUseCase{repo: repo}
}

func (uc *GetTournamentsUseCase) Execute(ctx context.Context) ([]*tournamentDomain.Tournament, error) {
	return uc.repo.GetAll(ctx)
}
