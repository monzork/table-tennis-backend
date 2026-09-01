package leaderboard

import "context"

// PreviousEloRepository is the narrow slice of event.ParticipantRepository
// this use case needs -- the player's Elo snapshot from their most recently
// finished event, used as the baseline for the ranking page's rank-movement
// indicator.
type PreviousEloRepository interface {
	GetPreviousEloSnapshots(ctx context.Context, rankType string) (map[string]int16, error)
}

type GetRankMovementUseCase struct {
	repo PreviousEloRepository
}

func NewGetRankMovementUseCase(repo PreviousEloRepository) *GetRankMovementUseCase {
	return &GetRankMovementUseCase{repo: repo}
}

// Execute returns a map of playerID -> Elo held immediately before that
// player's most recently finished event, for the given rank type
// ("singles"|"doubles"). Players with no finished event are absent.
func (uc *GetRankMovementUseCase) Execute(ctx context.Context, rankType string) (map[string]int16, error) {
	return uc.repo.GetPreviousEloSnapshots(ctx, rankType)
}
