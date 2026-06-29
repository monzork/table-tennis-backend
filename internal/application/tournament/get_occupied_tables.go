package tournament

import (
	"context"

	"table-tennis-backend/internal/domain/tournament"
)

type GetOccupiedTablesUseCase struct {
	matchRepo tournament.MatchRepository
}

func NewGetOccupiedTablesUseCase(matchRepo tournament.MatchRepository) *GetOccupiedTablesUseCase {
	return &GetOccupiedTablesUseCase{
		matchRepo: matchRepo,
	}
}

func (uc *GetOccupiedTablesUseCase) Execute(ctx context.Context, t *tournament.Tournament) ([]int, error) {
	var occupiedList []int
	var err error
	if t != nil {
		if t.EventID != nil {
			eventUUID := *t.EventID
			occupiedList, err = uc.matchRepo.GetOccupiedTablesByEvent(ctx, eventUUID)
		} else {
			tourneyUUID := t.ID
			occupiedList, err = uc.matchRepo.GetOccupiedTablesByTournament(ctx, tourneyUUID)
		}
	}
	return occupiedList, err
}
