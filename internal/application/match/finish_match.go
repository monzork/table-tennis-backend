package match

import (
	"errors"
	event "table-tennis-backend/internal/domain/event"
)

type FinishMatchUseCase struct{}

func NewFinishMatchUseCase() *FinishMatchUseCase {
	return &FinishMatchUseCase{}
}

func (uc *FinishMatchUseCase) Execute(m *event.Match, winnerTeam string) error {
	if winnerTeam != "A" && winnerTeam != "B" {
		return errors.New("invalid winner team")
	}

	m.WinnerTeam = winnerTeam

	m.Status = "finished"
	return nil
}
