package player

import (
	"context"
	"table-tennis-backend/internal/domain/player"
)

// GetPlayerRankUseCase reports a player's position within the overall
// (singles or doubles) Elo ranking, e.g. "#12 of 156".
type GetPlayerRankUseCase struct {
	repo player.Repository
}

func NewGetPlayerRankUseCase(repo player.Repository) *GetPlayerRankUseCase {
	return &GetPlayerRankUseCase{repo: repo}
}

// Execute returns rank (1-based position, 0 if the player isn't found) and
// the total number of ranked players for rankType ("singles" or "doubles").
func (uc *GetPlayerRankUseCase) Execute(ctx context.Context, playerID, rankType string) (rank int, total int, err error) {
	var players []*player.Player
	if rankType == "doubles" {
		players, err = uc.repo.GetAllDoubles(ctx)
	} else {
		players, err = uc.repo.GetAllSingles(ctx)
	}
	if err != nil {
		return 0, 0, err
	}

	total = len(players)
	for i, p := range players {
		if p.ID == playerID {
			return i + 1, total, nil
		}
	}
	return 0, total, nil
}
