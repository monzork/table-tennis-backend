package tournament

import (
	"context"
	"github.com/google/uuid"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

type MovePlayerUseCase struct {
	repo *bun.TournamentRepository
}

func NewMovePlayerUseCase(repo *bun.TournamentRepository) *MovePlayerUseCase {
	return &MovePlayerUseCase{repo: repo}
}

func (uc *MovePlayerUseCase) Execute(ctx context.Context, tournamentIDStr, playerIDStr, targetGroupIDStr string, targetIndex int) error {
	tID, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		return err
	}
	pID, err := uuid.Parse(playerIDStr)
	if err != nil {
		return err
	}
	var targetGroupID uuid.UUID
	if targetGroupIDStr != "" {
		var err error
		targetGroupID, err = uuid.Parse(targetGroupIDStr)
		if err != nil {
			return err
		}
	}

	t, err := uc.repo.GetByID(ctx, tID)
	if err != nil {
		return err
	}

	if err := t.MovePlayer(pID, targetGroupID, targetIndex); err != nil {
		return err
	}

	return uc.repo.Update(ctx, t)
}
