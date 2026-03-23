package match

import (
	"context"
	player "table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
	bunDB "github.com/uptrace/bun"
)

type MatchView struct {
	ID           uuid.UUID
	TournamentID uuid.UUID
	MatchType    string
	TeamA        []*player.Player
	TeamB        []*player.Player
	Status       string
	WinnerTeam   *string
}

type GetMatchesUseCase struct {
	db         *bunDB.DB
	playerRepo bun.PlayerRepository
}

func NewGetMatchesUseCase(db bunDB.DB, playerRepo bun.PlayerRepository) *GetMatchesUseCase {
	return &GetMatchesUseCase{
		db:         &db,
		playerRepo: playerRepo,
	}
}

// Fetch all matches as view models
func (r *GetMatchesUseCase) GetAllViews(ctx context.Context) ([]*MatchView, error) {
	var models []bun.MatchModel
	if err := r.db.NewSelect().Model(&models).Order("created_at DESC").Scan(ctx); err != nil {
		return nil, err
	}

	views := make([]*MatchView, 0, len(models))
	for _, m := range models {
		teamA := []*player.Player{}
		if p, err := r.playerRepo.GetById(ctx, m.TeamAPlayer1ID); err == nil {
			teamA = append(teamA, p)
		}
		if m.TeamAPlayer2ID != nil {
			if p, err := r.playerRepo.GetById(ctx, *m.TeamAPlayer2ID); err == nil {
				teamA = append(teamA, p)
			}
		}

		teamB := []*player.Player{}
		if p, err := r.playerRepo.GetById(ctx, m.TeamBPlayer1ID); err == nil {
			teamB = append(teamB, p)
		}
		if m.TeamBPlayer2ID != nil {
			if p, err := r.playerRepo.GetById(ctx, *m.TeamBPlayer2ID); err == nil {
				teamB = append(teamB, p)
			}
		}

		views = append(views, &MatchView{
			ID:           m.ID,
			TournamentID: m.TournamentID,
			MatchType:    m.MatchType,
			TeamA:        teamA,
			TeamB:        teamB,
			Status:       m.Status,
			WinnerTeam:   m.WinnerTeam,
		})
	}
	return views, nil
}
