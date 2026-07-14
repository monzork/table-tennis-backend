package event

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

func (t *Team) AverageElo(tournamentType string) int16 {
	avgElo := int16(1000)
	if len(t.Players) > 0 {
		sum := int32(0)
		for _, p := range t.Players {
			if tournamentType == "doubles" || tournamentType == "mixed_doubles" {
				sum += int32(p.DoublesElo)
			} else {
				sum += int32(p.SinglesElo)
			}
		}
		avgElo = int16(sum / int32(len(t.Players)))
	}
	return avgElo
}
