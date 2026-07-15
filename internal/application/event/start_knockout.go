package event

import (
	"context"
	"fmt"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
)

type StartKnockoutStageUseCase struct {
	repo      event.Repository
	matchRepo event.MatchRepository
}

func NewStartKnockoutStageUseCase(repo event.Repository, matchRepo event.MatchRepository) *StartKnockoutStageUseCase {
	return &StartKnockoutStageUseCase{
		repo:      repo,
		matchRepo: matchRepo,
	}
}

func (uc *StartKnockoutStageUseCase) Execute(ctx context.Context, tournamentID, divID string, firstRoundMatches []event.Match) error {
	t, err := uc.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}

	if t.Status == "finished" {
		return fmt.Errorf("cannot start knockout stage for a finished event")
	}

	// Insert all the first round matches that don't already exist
	for _, m := range firstRoundMatches {
		exists := false
		for _, exM := range t.Matches {
			if exM.ID == m.ID || (exM.DivisionID == divID && exM.Stage == m.Stage && exM.RoundNumber == m.RoundNumber && sameTeams(exM, m)) {
				exists = true
				break
			}
		}

		if !exists {
			if m.ID == "" {
				m.ID = idgen.Generate()
			}
			m.Status = "scheduled"
			if err := uc.matchRepo.Save(ctx, &m); err != nil {
				return err
			}
		}
	}

	return nil
}

func sameTeams(m1, m2 event.Match) bool {
	if len(m1.TeamA) != len(m2.TeamA) || len(m1.TeamB) != len(m2.TeamB) {
		return false
	}
	// Just check the first player for simplicity
	if len(m1.TeamA) > 0 && m1.TeamA[0].ID != m2.TeamA[0].ID {
		return false
	}
	if len(m1.TeamB) > 0 && m1.TeamB[0].ID != m2.TeamB[0].ID {
		return false
	}
	return true
}
