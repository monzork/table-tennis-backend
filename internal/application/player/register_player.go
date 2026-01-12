package player

import (
	"context"
	playerDomain "table-tennis-backend/internal/domain/player"
	playerDB "table-tennis-backend/internal/infrastructure/persistence/bun"

	"time"
)

type RegisterPlayerUseCase struct {
	repo *playerDB.PlayerRepository
}

func NewRegisterPlayerUseCase(repo *playerDB.PlayerRepository) *RegisterPlayerUseCase {
	return &RegisterPlayerUseCase{repo: repo}
}

func (uc *RegisterPlayerUseCase) Execute(ctx context.Context, firstName, lastName string, birthdate string, country string) (*playerDomain.Player, error) {
	bd, err := time.Parse("2006-01-02", birthdate)
	if err != nil {
		return nil, err
	}

	p, err := playerDomain.NewPlayer(firstName, lastName, bd, country)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}
