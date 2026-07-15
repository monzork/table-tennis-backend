package match

import (
	"context"
	"errors"
	"strconv"
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
func (uc *StartMatchUseCase) Execute(ctx context.Context, matchID string, manualTableStr string) (*event.Match, error) {
	m, err := uc.matchRepo.GetByID(ctx, matchID)
	if err != nil {
		return nil, errors.New("Match not found")
	}

	t, err := uc.eventRepo.GetByID(ctx, m.TournamentID)
	if err != nil {
		return nil, err
	}

	// Manual override
	if manualTableStr != "" {
		if tNum, err := strconv.Atoi(manualTableStr); err == nil {
			isOccupied, _ := uc.matchRepo.IsTableOccupiedByOtherMatch(ctx, m.ID, tNum)
			if isOccupied {
				return nil, errors.New("Table is occupied")
			}
			m.TableNumber = &tNum
		}
	}

	// Auto-assign table logic if missing
	if m.TableNumber == nil {
		var eventNumTables int
		if t.EventID != nil {
			eventNumTables, _ = uc.eventRepo.GetEventNumTables(ctx, *t.EventID)
		}

		totalTables := 4
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
			return nil, errors.New("No tables available")
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

	return m, nil
}
