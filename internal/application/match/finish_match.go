package match

import (
	match "table-tennis-backend/internal/domain/match"
	"table-tennis-backend/internal/domain/player"
	tournament "table-tennis-backend/internal/domain/tournament"

	"github.com/google/uuid"
)

type FinishMatchUseCase struct{}

func NewFinishMatchUseCase() *FinishMatchUseCase {
	return &FinishMatchUseCase{}
}

func (uc *FinishMatchUseCase) Execute(m *tournament.Match, winnerID uuid.UUID) error {
	for _, p := range m.Players {
		if p.ID == winnerID {
			m.Winner = p
			break
		}
	}

	if m.Winner == nil {
		return player.ErrInvalidName
	}

	// Update ELO
	if len(m.Players) == 2 {
		newElo1, newElo2 := match.CalculateElo(m.Players[0], m.Players[1], m.Winner)
		m.Players[0].UpdateElo(newElo1)
		m.Players[1].UpdateElo(newElo2)
	}

	m.Status = "finished"
	return nil
}
