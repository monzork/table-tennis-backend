package match

import (
	"errors"
	match "table-tennis-backend/internal/domain/match"
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

	// Calculate and directly apply Elo
	match.CalculateAndApplyElo(m.MatchType, m.TeamA, m.TeamB, m.WinnerTeam)

	m.Status = "finished"
	return nil
}
