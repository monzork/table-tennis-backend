package player

import (
	"context"
	playerDomain "table-tennis-backend/internal/domain/player"

	"github.com/google/uuid"
	"time"
)

type RegisterPlayerUseCase struct {
	repo playerDomain.Repository
}

func NewRegisterPlayerUseCase(repo playerDomain.Repository) *RegisterPlayerUseCase {
	return &RegisterPlayerUseCase{repo: repo}
}

func (uc *RegisterPlayerUseCase) Execute(ctx context.Context, firstName, lastName string, birthdate, gender, country, department, whatsAppNumber string, singlesElo, doublesElo int16) (*playerDomain.Player, error) {
	bd, err := time.Parse("2006-01-02", birthdate)
	if err != nil {
		return nil, err
	}

	p, err := playerDomain.NewPlayer(uuid.NewString(), firstName, lastName, bd, gender, country, department)
	if err != nil {
		return nil, err
	}

	p.WhatsAppNumber = whatsAppNumber

	if singlesElo > 0 {
		p.UpdateSinglesElo(singlesElo)
	}
	if doublesElo > 0 {
		p.UpdateDoublesElo(doublesElo)
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}
