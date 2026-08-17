package account

import (
	"context"

	"table-tennis-backend/internal/domain/player"
)

// GetLinkedPlayersUseCase wraps player.Repository.GetByGuardianAccountID —
// the players (children, or an admin-linked adult) an account may manage.
type GetLinkedPlayersUseCase struct {
	playerRepo player.Repository
}

func NewGetLinkedPlayersUseCase(playerRepo player.Repository) *GetLinkedPlayersUseCase {
	return &GetLinkedPlayersUseCase{playerRepo: playerRepo}
}

func (uc *GetLinkedPlayersUseCase) Execute(ctx context.Context, accountID string) ([]*player.Player, error) {
	return uc.playerRepo.GetByGuardianAccountID(ctx, accountID)
}
