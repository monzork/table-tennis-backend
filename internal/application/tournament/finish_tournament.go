package tournament

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"table-tennis-backend/internal/domain/match"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

type FinishTournamentUseCase struct {
	tournamentRepo *bun.TournamentRepository
	matchRepo      *bun.MatchRepository
	playerRepo     *bun.PlayerRepository
}

func NewFinishTournamentUseCase(
	tournamentRepo *bun.TournamentRepository,
	matchRepo *bun.MatchRepository,
	playerRepo *bun.PlayerRepository,
) *FinishTournamentUseCase {
	return &FinishTournamentUseCase{
		tournamentRepo: tournamentRepo,
		matchRepo:      matchRepo,
		playerRepo:     playerRepo,
	}
}

func (uc *FinishTournamentUseCase) Execute(ctx context.Context, tournamentID uuid.UUID) error {
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}
	if t.Status == "finished" {
		return errors.New("tournament is already finished")
	}

	// Fetch all finished matches for the tournament in chronological order
	var matchModels []bun.MatchModel
	err = uc.matchRepo.DB().NewSelect().
		Model(&matchModels).
		Where("tournament_id = ?", tournamentID).
		Where("status = ?", "finished").
		Order("updated_at ASC").
		Scan(ctx)
	if err != nil {
		return err
	}

	// Apply Elo changes match by match in order
	for _, m := range matchModels {
		if m.WinnerTeam == nil || *m.WinnerTeam == "" {
			continue
		}

		var teamA, teamB []*player.Player
		
		p1a, err1 := uc.playerRepo.GetById(ctx, m.TeamAPlayer1ID)
		if err1 == nil && p1a != nil {
			teamA = append(teamA, p1a)
		}
		if m.TeamAPlayer2ID != nil {
			p2a, err2 := uc.playerRepo.GetById(ctx, *m.TeamAPlayer2ID)
			if err2 == nil && p2a != nil {
				teamA = append(teamA, p2a)
			}
		}

		p1b, err3 := uc.playerRepo.GetById(ctx, m.TeamBPlayer1ID)
		if err3 == nil && p1b != nil {
			teamB = append(teamB, p1b)
		}
		if m.TeamBPlayer2ID != nil {
			p2b, err4 := uc.playerRepo.GetById(ctx, *m.TeamBPlayer2ID)
			if err4 == nil && p2b != nil {
				teamB = append(teamB, p2b)
			}
		}

		if len(teamA) > 0 && len(teamB) > 0 {
			match.CalculateAndApplyElo(m.MatchType, teamA, teamB, *m.WinnerTeam)
			for _, p := range teamA {
				_ = uc.playerRepo.Save(ctx, p)
			}
			for _, p := range teamB {
				_ = uc.playerRepo.Save(ctx, p)
			}
		}
	}

	// Finalize EloAfter snapshots
	for _, p := range t.Participants {
		if updatedPlayer, err := uc.playerRepo.GetById(ctx, p.ID); err == nil {
			_, _ = uc.matchRepo.DB().NewUpdate().
				TableExpr("tournament_participants").
				Set("elo_after_singles = ?, elo_after_doubles = ?", updatedPlayer.SinglesElo, updatedPlayer.DoublesElo).
				Where("tournament_id = ? AND player_id = ?", tournamentID, p.ID).
				Exec(ctx)
		}
	}

	// Mark tournament as finished
	t.Status = "finished"
	return uc.tournamentRepo.Update(ctx, t)
}
