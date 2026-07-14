package match

import (
	"context"

	"table-tennis-backend/internal/domain/event"
)

type AutoAssignTablesUseCase struct {
	matchRepo      event.MatchRepository
}

func NewAutoAssignTablesUseCase(matchRepo event.MatchRepository) *AutoAssignTablesUseCase {
	return &AutoAssignTablesUseCase{
		matchRepo:      matchRepo,
	}
}

func (uc *AutoAssignTablesUseCase) Execute(ctx context.Context, tournamentID string) ([]event.Match, error) {
	// Dummy implementation for now to fix compile errors
	return []event.Match{}, nil
}
