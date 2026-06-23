package tournament

import (
	"errors"
	"table-tennis-backend/internal/domain/player"
)

type Team struct {
	ID           string
	TournamentID string
	Name         string
	Players      []*player.Player
}

func NewTeam(id string, tournamentID string, name string) (*Team, error) {
	if name == "" {
		return nil, errors.New("team name cannot be empty")
	}
	return &Team{
		ID:           id,
		TournamentID: tournamentID,
		Name:         name,
		Players:      []*player.Player{},
	}, nil
}
