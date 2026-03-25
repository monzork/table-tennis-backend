package division

import (
	"context"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

type DivisionUseCase struct {
	repo *bun.DivisionRepository
}

func NewDivisionUseCase(repo *bun.DivisionRepository) *DivisionUseCase {
	return &DivisionUseCase{repo: repo}
}

func (uc *DivisionUseCase) GetAll(ctx context.Context) ([]*division.Division, error) {
	return uc.repo.GetAll(ctx)
}

func (uc *DivisionUseCase) Save(ctx context.Context, id, name string, displayOrder int, minElo int16, maxElo *int16, category, color string) error {
	var d *division.Division
	var err error

	if id != "" {
		// Try to fetch existing
		d, err = uc.repo.GetById(ctx, id)
		if err != nil {
             // fallback to new if not found
			d, err = division.NewDivision(name, displayOrder, minElo, maxElo, category, color)
		} else {
			d.Name = name
			d.DisplayOrder = displayOrder
			d.MinElo = minElo
			if maxElo != nil && minElo >= *maxElo {
				return division.ErrInvalidEloRange
			}
			d.MaxElo = maxElo
			if category != "" {
				d.Category = category
			}
			if color != "" {
				d.Color = color
			}
		}
	} else {
		d, err = division.NewDivision(name, displayOrder, minElo, maxElo, category, color)
	}

	if err != nil {
		return err
	}

	return uc.repo.Save(ctx, d)
}

func (uc *DivisionUseCase) Delete(ctx context.Context, id string) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *DivisionUseCase) GetById(ctx context.Context, id string) (*division.Division, error) {
	return uc.repo.GetById(ctx, id)
}
