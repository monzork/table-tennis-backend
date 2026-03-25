package bun

import (
	"context"

	"table-tennis-backend/internal/domain/division"

	"github.com/uptrace/bun"
)

type DivisionRepository struct {
	db *bun.DB
}

func NewDivisionRepository(db *bun.DB) *DivisionRepository {
	return &DivisionRepository{db: db}
}

func (r *DivisionRepository) Save(ctx context.Context, d *division.Division) error {
	model := &DivisionModel{
		ID:           d.ID,
		Name:         d.Name,
		DisplayOrder: d.DisplayOrder,
		MinElo:       d.MinElo,
		MaxElo:       d.MaxElo,
		Category:     d.Category,
		Color:        d.Color,
	}

	_, err := r.db.NewInsert().Model(model).On("CONFLICT (id) DO UPDATE").Exec(ctx)
	return err
}

func (r *DivisionRepository) GetAll(ctx context.Context) ([]*division.Division, error) {
	var models []DivisionModel
	err := r.db.NewSelect().Model(&models).Order("display_order ASC").Scan(ctx)

	if err != nil {
		return nil, err
	}

	divisions := make([]*division.Division, len(models))
	for i, m := range models {
		divisions[i] = &division.Division{
			ID:           m.ID,
			Name:         m.Name,
			DisplayOrder: m.DisplayOrder,
			MinElo:       m.MinElo,
			MaxElo:       m.MaxElo,
			Category:     m.Category,
			Color:        m.Color,
		}
	}
	return divisions, nil
}

func (r *DivisionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model((*DivisionModel)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *DivisionRepository) GetById(ctx context.Context, id string) (*division.Division, error) {
	var m DivisionModel
	err := r.db.NewSelect().Model(&m).Where("id = ?", id).Scan(ctx)

	if err != nil {
		return nil, err
	}

	return &division.Division{
		ID:           m.ID,
		Name:         m.Name,
		DisplayOrder: m.DisplayOrder,
		MinElo:       m.MinElo,
		MaxElo:       m.MaxElo,
		Category:     m.Category,
		Color:        m.Color,
	}, nil
}
