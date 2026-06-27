package tournament

import (
	"context"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

type MovePlayerUseCase struct {
	repo tournamentDomain.Repository
}

func NewMovePlayerUseCase(repo tournamentDomain.Repository) *MovePlayerUseCase {
	return &MovePlayerUseCase{repo: repo}
}

func (uc *MovePlayerUseCase) Execute(ctx context.Context, tournamentIDStr, playerIDStr, targetGroupIDStr string, targetIndex int) error {
	t, err := uc.repo.GetByID(ctx, tournamentIDStr)
	if err != nil {
		return err
	}

	if err := t.MovePlayer(playerIDStr, targetGroupIDStr, targetIndex); err != nil {
		return err
	}

	return uc.repo.UpdateGroups(ctx, t)
}
