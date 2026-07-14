package event

import (
	"context"
	tournamentDomain "table-tennis-backend/internal/domain/event"
)

type GetTournamentsUseCase struct {
	repo tournamentDomain.Repository
}

func NewGetTournamentsUseCase(repo tournamentDomain.Repository) *GetTournamentsUseCase {
	return &GetTournamentsUseCase{repo: repo}
}

func (uc *GetTournamentsUseCase) Execute(ctx context.Context) ([]*tournamentDomain.Event, error) {
	return uc.repo.GetAll(ctx)
}
