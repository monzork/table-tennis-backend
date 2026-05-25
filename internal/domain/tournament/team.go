package tournament

import (
	"errors"
	"table-tennis-backend/internal/domain/player"

	"github.com/google/uuid"
)

type Team struct {
	ID           uuid.UUID
	TournamentID uuid.UUID
	Name         string
	Players      []*player.Player
}

func NewTeam(tournamentID uuid.UUID, name string) (*Team, error) {
	if name == "" {
		return nil, errors.New("team name cannot be empty")
	}
	return &Team{
		ID:           uuid.New(),
		TournamentID: tournamentID,
		Name:         name,
		Players:      []*player.Player{},
	}, nil
}
