package bun

import (
	"context"
	"database/sql"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TournamentRepository struct {
	db *bun.DB
}

func NewTournamentRepository(db *bun.DB) *TournamentRepository {
	return &TournamentRepository{db: db}
}

func (r *TournamentRepository) Save(ctx context.Context, t *tournament.Tournament) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	model := &TournamentModel{
		ID:        t.ID,
		Name:      t.Name,
		Type:      t.Type,
		Format:    t.Format,
		StartDate: t.StartDate,
		EndDate:   t.EndDate,
	}
	_, err = tx.NewInsert().Model(model).Exec(ctx)
	if err != nil {
		return err
	}

	// Save participants
	for _, p := range t.Participants {
		partModel := &TournamentParticipantModel{
			TournamentID: t.ID,
			PlayerID:     p.ID,
		}
		_, err = tx.NewInsert().Model(partModel).Exec(ctx)
		if err != nil {
			return err
		}
	}

	// Save groups
	for _, g := range t.Groups {
		groupModel := &GroupModel{
			ID:           g.ID,
			TournamentID: t.ID,
			Name:         g.Name,
		}
		_, err = tx.NewInsert().Model(groupModel).Exec(ctx)
		if err != nil {
			return err
		}

		// Save group participants
		for _, p := range g.Players {
			gpModel := &GroupParticipantModel{
				GroupID:  g.ID,
				PlayerID: p.ID,
			}
			_, err = tx.NewInsert().Model(gpModel).Exec(ctx)
			if err != nil {
				return err
			}
		}
	}

	// Save default stage rules
	if err := saveStageRules(ctx, tx, t.StageRules); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *TournamentRepository) GetAll(ctx context.Context) ([]*tournament.Tournament, error) {
	var models []TournamentModel
	if err := r.db.NewSelect().Model(&models).Scan(ctx); err != nil {
		return nil, err
	}
	tournaments := make([]*tournament.Tournament, len(models))
	for i, m := range models {
		tournaments[i] = &tournament.Tournament{
			ID:        m.ID,
			Name:      m.Name,
			Type:      m.Type,
			Format:    m.Format,
			StartDate: m.StartDate,
			EndDate:   m.EndDate,
			Rules:     []tournament.Rule{},
			Matches:   []tournament.Match{},
		}
	}
	return tournaments, nil
}

func (r *TournamentRepository) GetByID(ctx context.Context, id uuid.UUID) (*tournament.Tournament, error) {
	model := new(TournamentModel)
	err := r.db.NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}

	// Load participants
	var partModels []TournamentParticipantModel
	_ = r.db.NewSelect().Model(&partModels).Where("tournament_id = ?", id).Scan(ctx)

	var participantPlayers []*player.Player
	for _, pt := range partModels {
		var pm PlayerModel
		if e := r.db.NewSelect().Model(&pm).Where("id = ?", pt.PlayerID).Scan(ctx); e == nil {
			participantPlayers = append(participantPlayers, &player.Player{
				ID:         pm.ID,
				FirstName:  pm.FirstName,
				LastName:   pm.LastName,
				SinglesElo: pm.SinglesElo,
				DoublesElo: pm.DoublesElo,
				Country:    pm.Country,
			})
		}
	}

	// Load groups with their players
	var groupModels []GroupModel
	_ = r.db.NewSelect().Model(&groupModels).Where("tournament_id = ?", id).Order("name ASC").Scan(ctx)

	var groups []tournament.Group
	for _, gm := range groupModels {
		var gpModels []GroupParticipantModel
		_ = r.db.NewSelect().Model(&gpModels).Where("group_id = ?", gm.ID).Scan(ctx)

		var groupPlayers []*player.Player
		for _, gp := range gpModels {
			var pm PlayerModel
			if e := r.db.NewSelect().Model(&pm).Where("id = ?", gp.PlayerID).Scan(ctx); e == nil {
				groupPlayers = append(groupPlayers, &player.Player{
					ID:         pm.ID,
					FirstName:  pm.FirstName,
					LastName:   pm.LastName,
					SinglesElo: pm.SinglesElo,
					DoublesElo: pm.DoublesElo,
					Country:    pm.Country,
				})
			}
		}
		groups = append(groups, tournament.Group{
			ID:      gm.ID,
			Name:    gm.Name,
			Players: groupPlayers,
		})
	}

	var matchModels []MatchModel
	if err := r.db.NewSelect().Model(&matchModels).Where("tournament_id = ?", id).Scan(ctx); err != nil && err != sql.ErrNoRows {
		// Just log or ignore if matches fail to load; it shouldn't fail the tournament
	}
	var matches []tournament.Match
	for _, mm := range matchModels {
		wt := ""
		if mm.WinnerTeam != nil {
			wt = *mm.WinnerTeam
		}
		m := tournament.Match{
			ID:         mm.ID,
			Status:     mm.Status,
			WinnerTeam: wt,
			TeamA:      []*player.Player{{ID: mm.TeamAPlayer1ID}},
			TeamB:      []*player.Player{{ID: mm.TeamBPlayer1ID}},
		}
		matches = append(matches, m)
	}

	return &tournament.Tournament{
		ID:           model.ID,
		Name:         model.Name,
		Type:         model.Type,
		Format:       model.Format,
		StartDate:    model.StartDate,
		EndDate:      model.EndDate,
		Participants: participantPlayers,
		Groups:       groups,
		Rules:        []tournament.Rule{},
		StageRules:   loadStageRules(ctx, r.db, model.ID),
		Matches:      matches,
	}, nil
}

func (r *TournamentRepository) Update(ctx context.Context, t *tournament.Tournament) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	model := &TournamentModel{
		ID:        t.ID,
		Name:      t.Name,
		Type:      t.Type,
		Format:    t.Format,
		StartDate: t.StartDate,
		EndDate:   t.EndDate,
	}

	_, err = tx.NewUpdate().Model(model).WherePK().Exec(ctx)
	if err != nil {
		return err
	}

	// Scrub existing groups and participants
	tx.NewDelete().TableExpr("group_participants").Where("group_id IN (SELECT id FROM groups WHERE tournament_id = ?)", t.ID).Exec(ctx)
	tx.NewDelete().TableExpr("groups").Where("tournament_id = ?", t.ID).Exec(ctx)
	tx.NewDelete().TableExpr("tournament_participants").Where("tournament_id = ?", t.ID).Exec(ctx)

	// Refresh participants
	for _, p := range t.Participants {
		partModel := &TournamentParticipantModel{
			TournamentID: t.ID,
			PlayerID:     p.ID,
		}
		_, err = tx.NewInsert().Model(partModel).Exec(ctx)
		if err != nil {
			return err
		}
	}

	// Refresh groups
	for _, g := range t.Groups {
		groupModel := &GroupModel{
			ID:           g.ID,
			TournamentID: t.ID,
			Name:         g.Name,
		}
		_, err = tx.NewInsert().Model(groupModel).Exec(ctx)
		if err != nil {
			return err
		}

		for _, p := range g.Players {
			gpModel := &GroupParticipantModel{
				GroupID:  g.ID,
				PlayerID: p.ID,
			}
			_, err = tx.NewInsert().Model(gpModel).Exec(ctx)
			if err != nil {
				return err
			}
		}
	}

	// Replace stage rules if changed
	if len(t.StageRules) > 0 {
		if err := replaceStageRules(ctx, tx, t.ID, t.StageRules); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *TournamentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Manual cascade since SQLite FK cascade may not be enabled
	tx.NewDelete().TableExpr("group_participants").Where("group_id IN (SELECT id FROM groups WHERE tournament_id = ?)", id).Exec(ctx)
	tx.NewDelete().TableExpr("groups").Where("tournament_id = ?", id).Exec(ctx)
	tx.NewDelete().TableExpr("tournament_participants").Where("tournament_id = ?", id).Exec(ctx)
	_, err = tx.NewDelete().Model(&TournamentModel{}).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return err
	}
	return tx.Commit()
}
