package match

import (
	"context"
	player "table-tennis-backend/internal/domain/player"
	tournament "table-tennis-backend/internal/domain/tournament"
)

type MatchView struct {
	ID           string
	TournamentID string
	MatchType    string
	TeamA        []*player.Player
	TeamB        []*player.Player
	Status       string
	WinnerTeam   *string
}

type GetMatchesUseCase struct {
	matchRepo tournament.MatchRepository
}

func NewGetMatchesUseCase(matchRepo tournament.MatchRepository) *GetMatchesUseCase {
	return &GetMatchesUseCase{
		matchRepo: matchRepo,
	}
}

// Fetch all matches as view models
func (uc *GetMatchesUseCase) GetAllViews(ctx context.Context) ([]*MatchView, error) {
	matches, err := uc.matchRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]*MatchView, 0, len(matches))
	for _, m := range matches {
		var winnerTeam *string
		if m.WinnerTeam != "" {
			w := m.WinnerTeam
			winnerTeam = &w
		}
		views = append(views, &MatchView{
			ID:           m.ID,
			TournamentID: m.TournamentID,
			MatchType:    m.MatchType,
			TeamA:        m.TeamA,
			TeamB:        m.TeamB,
			Status:       m.Status,
			WinnerTeam:   winnerTeam,
		})
	}
	return views, nil
}
