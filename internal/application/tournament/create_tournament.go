package tournament

import (
	"context"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	"time"

	"github.com/google/uuid"
)

type CreateTournamentUseCase struct {
	repo       *bun.TournamentRepository
	playerRepo *bun.PlayerRepository
}

func NewCreateTournamentUseCase(repo *bun.TournamentRepository, playerRepo *bun.PlayerRepository) *CreateTournamentUseCase {
	return &CreateTournamentUseCase{repo: repo, playerRepo: playerRepo}
}

type NewPlayerData struct {
	FirstName string
	LastName  string
	Gender    string
}

func (uc *CreateTournamentUseCase) Execute(
	ctx context.Context,
	name string,
	tournamentType string,
	format string,
	startStr, endStr string,
	participantIDs []string,
	newPlayers []NewPlayerData,
) (*tournamentDomain.Tournament, error) {
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
	for _, idStr := range participantIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		p, err := uc.playerRepo.GetById(ctx, id)
		if err == nil {
			participants = append(participants, p)
		}
	}

	// Handle new players
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

	if err := uc.repo.Save(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}
