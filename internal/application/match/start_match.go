package match

import (
	"context"
	"errors"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/tournament"
)

type StartMatchUseCase struct {
	matchRepo      event.MatchRepository
	eventRepo      event.Repository
	tournamentRepo tournament.Repository
	createUC       *CreateMatchUseCase
}

func NewStartMatchUseCase(matchRepo event.MatchRepository, eventRepo event.Repository, tournamentRepo tournament.Repository, createUC *CreateMatchUseCase) *StartMatchUseCase {
	return &StartMatchUseCase{
		matchRepo:      matchRepo,
		eventRepo:      eventRepo,
		tournamentRepo: tournamentRepo,
		createUC:       createUC,
	}
}

// StartMatch assigns a table (if not provided) and starts the match.
func (uc *StartMatchUseCase) Execute(ctx context.Context, cmd event.StartMatchCommand) (*event.StartMatchResult, error) {
	m, err := uc.matchRepo.GetByID(ctx, cmd.MatchID)
	if err != nil {
		return nil, errors.New("Match not found")
	}

	t, err := uc.eventRepo.GetByID(ctx, m.TournamentID)
	if err != nil {
		return nil, err
	}

	// Manual override
	if cmd.TableNumber != nil {
		isOccupied, _ := uc.matchRepo.IsTableOccupiedByOtherMatch(ctx, m.ID, *cmd.TableNumber)
		if isOccupied {
			return nil, event.ErrTableOccupied
		}
		m.TableNumber = cmd.TableNumber
	}

	// Auto-assign table logic if missing
	if m.TableNumber == nil {
		var eventNumTables int
		if t.EventID != nil {
			eventNumTables, _ = uc.eventRepo.GetEventNumTables(ctx, *t.EventID)
		}

		totalTables := cmd.TotalTables
		if totalTables <= 0 {
			totalTables = 4
		}
		if t.NumTables > 0 {
			totalTables = t.NumTables
		}
		if eventNumTables > 0 {
			totalTables = eventNumTables
		}

		var occupiedList []int
		if t.EventID != nil {
			occupiedList, _ = uc.matchRepo.GetOccupiedTablesByEvent(ctx, *t.EventID)
		} else {
			occupiedList, _ = uc.matchRepo.GetOccupiedTablesByTournament(ctx, t.ID)
		}

		occupiedMap := make(map[int]bool)
		for _, num := range occupiedList {
			occupiedMap[num] = true
		}

		var availableTables []int
		for i := 1; i <= totalTables; i++ {
			if !occupiedMap[i] {
				availableTables = append(availableTables, i)
			}
		}

		if len(availableTables) == 0 {
			return nil, event.ErrTableOccupied
		}

		assignedTable := availableTables[0]

		priorityFound := false
		if m.DivisionID != "" {
			// Fetch the parent grand tournament to get table priorities
			if t.EventID != nil {
				if grandTourney, err := uc.tournamentRepo.GetByIDDeep(ctx, *t.EventID); err == nil && grandTourney.TablePriorities != nil {
					if priorities, ok := grandTourney.TablePriorities[m.DivisionID]; ok {
						for _, pTable := range priorities {
							if !occupiedMap[pTable] {
								assignedTable = pTable
								priorityFound = true
								break
							}
						}
					}
				}
			}
		}

		// Fallback to highest available if no priority match found
		if !priorityFound {
			if cmd.IsHighPriority {
				if !occupiedMap[1] {
					assignedTable = 1
				} else if !occupiedMap[2] && totalTables >= 2 {
					assignedTable = 2
				} else {
					assignedTable = availableTables[0]
				}
			} else {
				found := false
				for _, tbl := range availableTables {
					if tbl >= 3 {
						if !found || tbl > assignedTable {
							assignedTable = tbl
							found = true
						}
					}
				}
				if !found {
					if !occupiedMap[2] && totalTables >= 2 {
						assignedTable = 2
					} else if !occupiedMap[1] {
						assignedTable = 1
					} else {
						assignedTable = availableTables[0]
					}
				}
			}
		}

		m.TableNumber = &assignedTable
	}

	m.Status = "in_progress"
	err = uc.matchRepo.Save(ctx, m)
	if err != nil {
		return nil, err
	}

	pAName, pBName := "TBD", "TBD"
	if len(m.TeamA) > 0 {
		pAName = m.TeamA[0].FullName()
	}
	if len(m.TeamB) > 0 {
		pBName = m.TeamB[0].FullName()
	}
	tableNumber := 0
	if m.TableNumber != nil {
		tableNumber = *m.TableNumber
	}

	return &event.StartMatchResult{
		TableNumber: tableNumber,
		Pin:         m.Pin,
		PlayerAName: pAName,
		PlayerBName: pBName,
	}, nil
}
