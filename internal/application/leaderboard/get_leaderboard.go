package leaderboard

import (
	"context"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

type GetLeaderboardUseCase struct {
	playerRepo bun.PlayerRepository
}

func NewGetLeaderboardUseCase(repo bun.PlayerRepository) *GetLeaderboardUseCase {
	return &GetLeaderboardUseCase{playerRepo: repo}
}

// Returns players ordered by ELO descending
func (uc *GetLeaderboardUseCase) Execute(ctx context.Context) ([]*player.Player, error) {
	return uc.playerRepo.GetAll(ctx)
}
