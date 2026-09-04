package leaderboard

import "context"

// PlacementBonusRepository is the narrow slice of event.Repository this use
// case needs: the durable podium-bonus record written by
// event.ParticipantRepository.SavePlacementResults when a tournament
// finishes (see event.PlacementRecord), scoped to the single most recently
// finished tournament.
type PlacementBonusRepository interface {
	GetLatestTournamentPlacementBonuses(ctx context.Context, rankType string) (map[string]float64, error)
}

type GetPlacementBonusUseCase struct {
	repo PlacementBonusRepository
}

func NewGetPlacementBonusUseCase(repo PlacementBonusRepository) *GetPlacementBonusUseCase {
	return &GetPlacementBonusUseCase{repo: repo}
}

// Execute returns playerID -> flat placement Elo bonus earned in the single
// most recently finished tournament, for the given rank type ("singles" |
// "doubles"). A player absent from that tournament, or who didn't finish on
// the podium, is absent from the map.
func (uc *GetPlacementBonusUseCase) Execute(ctx context.Context, rankType string) (map[string]float64, error) {
	return uc.repo.GetLatestTournamentPlacementBonuses(ctx, rankType)
}
