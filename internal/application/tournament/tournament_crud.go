package tournament

import (
	"context"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	"time"

	"github.com/google/uuid"
)

// ─── Get By ID ───────────────────────────────────────────────────────────────

type GetTournamentByIDUseCase struct {
	repo *bun.TournamentRepository
}

func NewGetTournamentByIDUseCase(repo *bun.TournamentRepository) *GetTournamentByIDUseCase {
	return &GetTournamentByIDUseCase{repo: repo}
}

func (uc *GetTournamentByIDUseCase) Execute(ctx context.Context, idStr string) (*tournamentDomain.Tournament, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	return uc.repo.GetByID(ctx, id)
}

// ─── Update ──────────────────────────────────────────────────────────────────

type UpdateTournamentUseCase struct {
	repo       *bun.TournamentRepository
	playerRepo *bun.PlayerRepository
}

func NewUpdateTournamentUseCase(repo *bun.TournamentRepository, playerRepo *bun.PlayerRepository) *UpdateTournamentUseCase {
	return &UpdateTournamentUseCase{repo: repo, playerRepo: playerRepo}
}

func (uc *UpdateTournamentUseCase) Execute(
	ctx context.Context, idStr, name, tournamentType, format, startStr, endStr string,
	participantIDs []string, newPlayers []NewPlayerData,
) (*tournamentDomain.Tournament, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return nil, err
	}

	var participants []*playerDomain.Player

	// Handle existing players
	for _, pidStr := range participantIDs {
		pid, err := uuid.Parse(pidStr)
		if err != nil {
			continue
		}
		p, err := uc.playerRepo.GetById(ctx, pid)
		if err == nil {
			participants = append(participants, p)
		}
	}

	// Handle newly created players ad-hoc
	for _, np := range newPlayers {
		p, err := playerDomain.NewPlayer(np.FirstName, np.LastName, time.Now(), np.Gender, "")
		if err != nil {
			return nil, err
		}
		if err := uc.playerRepo.Save(ctx, p); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	t, err := tournamentDomain.NewTournament(name, tournamentType, format, start, end, []tournamentDomain.Rule{}, participants)
	if err != nil {
		return nil, err
	}
	t.ID = id

	if err := uc.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// ─── Delete ──────────────────────────────────────────────────────────────────

type DeleteTournamentUseCase struct {
	repo *bun.TournamentRepository
}

func NewDeleteTournamentUseCase(repo *bun.TournamentRepository) *DeleteTournamentUseCase {
	return &DeleteTournamentUseCase{repo: repo}
}

func (uc *DeleteTournamentUseCase) Execute(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	return uc.repo.Delete(ctx, id)
}
