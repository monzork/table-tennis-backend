package match

import (
	"errors"
	tournament "table-tennis-backend/internal/domain/tournament"
)

type FinishMatchUseCase struct{}

func NewFinishMatchUseCase() *FinishMatchUseCase {
	return &FinishMatchUseCase{}
}

func (uc *FinishMatchUseCase) Execute(m *tournament.Match, winnerTeam string) error {
	if winnerTeam != "A" && winnerTeam != "B" {
		return errors.New("invalid winner team")
	}

	m.WinnerTeam = winnerTeam

	m.Status = "finished"
	return nil
}
