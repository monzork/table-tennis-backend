package bun

import (
	"context"
	"table-tennis-backend/internal/domain/player"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type PlayerRepository struct {
	db *bun.DB
}

func NewPlayerRepository(db *bun.DB) *PlayerRepository {
	return &PlayerRepository{db: db}
}

func (r *PlayerRepository) Save(ctx context.Context, p *player.Player) error {
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return err
	}
	model := &PlayerModel{
		ID:             id,
		FirstName:      p.FirstName,
		LastName:       p.LastName,
		Birthdate:      p.Birthdate,
		Gender:         p.Gender,
		SinglesElo:     p.SinglesElo,
		DoublesElo:     p.DoublesElo,
		Country:        p.Country,
		Department:     p.Department,
		WhatsAppNumber: p.WhatsAppNumber,
		Pin:            p.Pin,
		NationalID:     p.NationalID,
	}

	_, err = r.db.NewInsert().Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name, gender = EXCLUDED.gender, singles_elo = EXCLUDED.singles_elo, doubles_elo = EXCLUDED.doubles_elo, country = EXCLUDED.country, whatsapp_number = EXCLUDED.whatsapp_number, department = EXCLUDED.department, pin = EXCLUDED.pin, national_id = EXCLUDED.national_id").
		Exec(ctx)

	return err
}

func (r *PlayerRepository) GetAllSingles(ctx context.Context) ([]*player.Player, error) {
	var models []PlayerModel
	err := r.db.NewSelect().Model(&models).OrderBy("singles_elo", bun.OrderDesc).Scan(ctx)

	if err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

func (r *PlayerRepository) GetAllDoubles(ctx context.Context) ([]*player.Player, error) {
	var models []PlayerModel
	err := r.db.NewSelect().Model(&models).OrderBy("doubles_elo", bun.OrderDesc).Scan(ctx)

	if err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

func (r *PlayerRepository) GetAll(ctx context.Context) ([]*player.Player, error) {
	return r.GetAllSingles(ctx)
}

func (r *PlayerRepository) GetSinglesByGender(ctx context.Context, gender string) ([]*player.Player, error) {
	var models []PlayerModel
	q := r.db.NewSelect().Model(&models).OrderBy("singles_elo", bun.OrderDesc)
	if gender != "" {
		q = q.Where("gender = ?", gender)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

func (r *PlayerRepository) GetDoublesByGender(ctx context.Context, gender string) ([]*player.Player, error) {
	var models []PlayerModel
	q := r.db.NewSelect().Model(&models).OrderBy("doubles_elo", bun.OrderDesc)
	if gender != "" {
		q = q.Where("gender = ?", gender)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

func (r *PlayerRepository) mapModelsToDomain(models []PlayerModel) []*player.Player {
	players := make([]*player.Player, len(models))
	for i, m := range models {
		players[i] = &player.Player{
			ID:             m.ID.String(),
			FirstName:      m.FirstName,
			LastName:       m.LastName,
			Birthdate:      m.Birthdate,
			Gender:         m.Gender,
			SinglesElo:     m.SinglesElo,
			DoublesElo:     m.DoublesElo,
			Country:        m.Country,
			Department:     m.Department,
			WhatsAppNumber: m.WhatsAppNumber,
			Pin:            m.Pin,
			NationalID:     m.NationalID,
		}
	}
	return players
}

func (r *PlayerRepository) GetById(ctx context.Context, id string) (*player.Player, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	var model PlayerModel
	err = r.db.NewSelect().Model(&model).Where("id = ?", uid).Scan(ctx)

	if err != nil {
		return nil, err
	}

	return &player.Player{
		ID:             model.ID.String(),
		FirstName:      model.FirstName,
		LastName:       model.LastName,
		Birthdate:      model.Birthdate,
		Gender:         model.Gender,
		SinglesElo:     model.SinglesElo,
		DoublesElo:     model.DoublesElo,
		Country:        model.Country,
		Department:     model.Department,
		WhatsAppNumber: model.WhatsAppNumber,
		Pin:            model.Pin,
		NationalID:     model.NationalID,
	}, nil
}

func (r *PlayerRepository) GetByIDs(ctx context.Context, ids []string) ([]*player.Player, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	uids := make([]uuid.UUID, 0, len(ids))
	for _, idStr := range ids {
		if uid, err := uuid.Parse(idStr); err == nil {
			uids = append(uids, uid)
		}
	}
	if len(uids) == 0 {
		return nil, nil
	}
	var models []PlayerModel
	err := r.db.NewSelect().Model(&models).Where("id IN (?)", bun.In(uids)).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

func (r *PlayerRepository) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	_, err = r.db.NewDelete().Model((*PlayerModel)(nil)).Where("id = ?", uid).Exec(ctx)
	return err
}

func (r *PlayerRepository) Search(ctx context.Context, query string) ([]*player.Player, error) {
	var models []PlayerModel
	q := r.db.NewSelect().Model(&models).OrderBy("singles_elo", bun.OrderDesc)
	if query != "" {
		q = q.Where("first_name LIKE ? OR last_name LIKE ?", "%"+query+"%", "%"+query+"%")
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}
