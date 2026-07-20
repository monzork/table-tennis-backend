package event

import (
	"context"
	"fmt"

	"table-tennis-backend/internal/domain/bracket"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
)

type StartKnockoutStageUseCase struct {
	repo         event.Repository
	matchRepo    event.MatchRepository
	divisionRepo division.Repository
}

func NewStartKnockoutStageUseCase(repo event.Repository, matchRepo event.MatchRepository, divisionRepo division.Repository) *StartKnockoutStageUseCase {
	return &StartKnockoutStageUseCase{
		repo:         repo,
		matchRepo:    matchRepo,
		divisionRepo: divisionRepo,
	}
}

func (uc *StartKnockoutStageUseCase) Execute(ctx context.Context, tournamentID, divID string) error {
	t, err := uc.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}

	if t.Status == "finished" {
		return fmt.Errorf("cannot start knockout stage for a finished event")
	}

	divisions, err := uc.divisionRepo.GetAll(ctx)
	if err != nil {
		return err
	}

	vm := bracket.BuildBracket(t, divisions, nil)

	var firstRoundMatches []event.Match
	for _, div := range vm.Divisions {
		if div.ID == divID && div.AllGroupsFinished && len(div.KnockoutBrackets) > 0 {
			for _, bracketBracket := range div.KnockoutBrackets {
				if err := bracket.ValidateSameGroupSeparation(div.Groups, bracketBracket.Advancing); err != nil {
					return err
				}

				if len(bracketBracket.Rounds) > 0 {
					r := bracketBracket.Rounds[0]
					// Only schedule matches from the very first round that have valid players
					for _, mv := range r.Matches {
						if mv.Player1 != nil && mv.Player2 != nil && mv.Player1.Player != nil && mv.Player2.Player != nil {
							if mv.Match == nil {
								firstRoundMatches = append(firstRoundMatches, event.Match{
									TournamentID: tournamentID,
									DivisionID:   divID,
									Stage:        mv.Stage,
									RoundNumber:  1,
									MatchType:    t.Type,
									TeamA:        []*player.Player{mv.Player1.Player},
									TeamB:        []*player.Player{mv.Player2.Player},
								})
							}
						}
					}
				}
			}
			break
		}
	}

	if len(firstRoundMatches) == 0 {
		return fmt.Errorf("no matches to start or group stage not fully finished")
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
