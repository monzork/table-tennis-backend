package tournament

import (
	"context"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

type GetTournamentsUseCase struct {
	repo *bun.TournamentRepository
}

func NewGetTournamentsUseCase(repo *bun.TournamentRepository) *GetTournamentsUseCase {
	return &GetTournamentsUseCase{repo: repo}
}

func (uc *GetTournamentsUseCase) Execute(ctx context.Context) ([]*tournamentDomain.Tournament, error) {
	return uc.repo.GetAll(ctx)
}
