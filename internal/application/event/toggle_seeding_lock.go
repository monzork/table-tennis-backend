package event

import (
	"context"
	tournamentDomain "table-tennis-backend/internal/domain/event"
)

type ToggleSeedingLockUseCase struct {
	repo tournamentDomain.Repository
}

func NewToggleSeedingLockUseCase(repo tournamentDomain.Repository) *ToggleSeedingLockUseCase {
	return &ToggleSeedingLockUseCase{repo: repo}
}

func (uc *ToggleSeedingLockUseCase) Execute(ctx context.Context, idStr string) error {
	t, err := uc.repo.GetByID(ctx, idStr)
	if err != nil {
		return err
	}

	t.ManualSeedingLocked = !t.ManualSeedingLocked

	return uc.repo.Update(ctx, t)
}
