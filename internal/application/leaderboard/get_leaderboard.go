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

// Returns players ordered by Elo descending based on type
func (uc *GetLeaderboardUseCase) Execute(ctx context.Context, rankingType string) ([]*player.Player, error) {
	if rankingType == "doubles" {
		return uc.playerRepo.GetAllDoubles(ctx)
	}
	return uc.playerRepo.GetAllSingles(ctx)
}

// ExecuteByGender returns players for given type filtered by gender ("M" or "F", "" = all)
func (uc *GetLeaderboardUseCase) ExecuteByGender(ctx context.Context, rankingType string, gender string) ([]*player.Player, error) {
	if rankingType == "doubles" {
		return uc.playerRepo.GetDoublesByGender(ctx, gender)
	}
	return uc.playerRepo.GetSinglesByGender(ctx, gender)
}

// For backward compatibility with Admin list
func (uc *GetLeaderboardUseCase) ExecuteSingles(ctx context.Context) ([]*player.Player, error) {
	return uc.playerRepo.GetAllSingles(ctx)
}
