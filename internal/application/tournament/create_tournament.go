package tournament

import (
	"context"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	"time"
)

type CreateTournamentUseCase struct {
	repo *bun.TournamentRepository
}

func NewCreateTournamentUseCase(repo *bun.TournamentRepository) *CreateTournamentUseCase {
	return &CreateTournamentUseCase{repo: repo}
}

func (uc *CreateTournamentUseCase) Execute(ctx context.Context, name string, startStr, endStr string) (*tournamentDomain.Tournament, error) {
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return nil, err
	}

	t, err := tournamentDomain.NewTournament(name, start, end, []tournamentDomain.Rule{})
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}
