package player

import (
	"context"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	"time"

	"github.com/google/uuid"
)

type UpdatePlayerUseCase struct {
	repo *bun.PlayerRepository
}

func NewUpdatePlayerUseCase(repo *bun.PlayerRepository) *UpdatePlayerUseCase {
	return &UpdatePlayerUseCase{repo: repo}
}

func (uc *UpdatePlayerUseCase) Execute(ctx context.Context, idStr, firstName, lastName, birthdate, gender, country, whatsAppNumber string, singlesElo, doublesElo int16) (*player.Player, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.GetById(ctx, id)
	if err != nil {
		return nil, err
	}

	if firstName != "" {
		p.FirstName = firstName
	}
	if lastName != "" {
		p.LastName = lastName
	}
	if birthdate != "" {
		if bd, err := time.Parse("2006-01-02", birthdate); err == nil {
			p.Birthdate = bd
		}
	}
	if gender != "" {
		p.Gender = gender
	}
	if country != "" {
		p.Country = country
	}
	
	p.WhatsAppNumber = whatsAppNumber
	
	p.UpdateSinglesElo(singlesElo)
	p.UpdateDoublesElo(doublesElo)

	err = uc.repo.Save(ctx, p)
	return p, err
}

type DeletePlayerUseCase struct {
	repo *bun.PlayerRepository
}

func NewDeletePlayerUseCase(repo *bun.PlayerRepository) *DeletePlayerUseCase {
	return &DeletePlayerUseCase{repo: repo}
}

func (uc *DeletePlayerUseCase) Execute(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	return uc.repo.Delete(ctx, id)
}
