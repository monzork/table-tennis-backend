package player

import (
	"context"
	"table-tennis-backend/internal/domain/player"
	"time"
)

type GetPlayerByIDUseCase struct {
	repo player.Repository
}

func NewGetPlayerByIDUseCase(repo player.Repository) *GetPlayerByIDUseCase {
	return &GetPlayerByIDUseCase{repo: repo}
}

func (uc *GetPlayerByIDUseCase) Execute(ctx context.Context, idStr string) (*player.Player, error) {
	return uc.repo.GetById(ctx, idStr)
}

type UpdatePlayerUseCase struct {
	repo player.Repository
}

func NewUpdatePlayerUseCase(repo player.Repository) *UpdatePlayerUseCase {
	return &UpdatePlayerUseCase{repo: repo}
}

func (uc *UpdatePlayerUseCase) Execute(ctx context.Context, idStr, firstName, lastName, birthdate, gender, country, department, whatsAppNumber string, singlesElo, doublesElo int16) (*player.Player, error) {
	p, err := uc.repo.GetById(ctx, idStr)
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
	p.Department = department
	
	p.WhatsAppNumber = whatsAppNumber
	
	p.UpdateSinglesElo(singlesElo)
	p.UpdateDoublesElo(doublesElo)

	err = uc.repo.Save(ctx, p)
	return p, err
}

type DeletePlayerUseCase struct {
	repo player.Repository
}

func NewDeletePlayerUseCase(repo player.Repository) *DeletePlayerUseCase {
	return &DeletePlayerUseCase{repo: repo}
}

func (uc *DeletePlayerUseCase) Execute(ctx context.Context, idStr string) error {
	return uc.repo.Delete(ctx, idStr)
}

type SearchPlayersUseCase struct {
	repo player.Repository
}

func NewSearchPlayersUseCase(repo player.Repository) *SearchPlayersUseCase {
	return &SearchPlayersUseCase{repo: repo}
}

func (uc *SearchPlayersUseCase) Execute(ctx context.Context, query string) ([]*player.Player, error) {
	return uc.repo.Search(ctx, query)
}
