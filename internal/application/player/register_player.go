package player

import (
	"context"
	"table-tennis-backend/internal/domain/idgen"
	playerDomain "table-tennis-backend/internal/domain/player"

	"time"
)

type RegisterPlayerUseCase struct {
	repo playerDomain.Repository
}

func NewRegisterPlayerUseCase(repo playerDomain.Repository) *RegisterPlayerUseCase {
	return &RegisterPlayerUseCase{repo: repo}
}

func (uc *RegisterPlayerUseCase) Execute(ctx context.Context, firstName, secondName, lastName, secondLastName string, birthdate, gender, country, department, whatsAppNumber, nationalID string, singlesElo, doublesElo int16) (*playerDomain.Player, error) {
	bd, err := time.Parse("2006-01-02", birthdate)
	if err != nil {
		return nil, err
	}

	p, err := playerDomain.NewPlayer(idgen.Generate(), firstName, lastName, bd, gender, country, department, nationalID)
	if err != nil {
		return nil, err
	}

	p.SecondName = secondName
	p.SecondLastName = secondLastName

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
