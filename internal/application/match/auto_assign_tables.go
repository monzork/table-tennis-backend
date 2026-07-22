package match

import (
	"context"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/tournament"
)

type AutoAssignTablesUseCase struct {
	matchRepo      event.MatchRepository
	tournamentRepo tournament.Repository
}

func NewAutoAssignTablesUseCase(matchRepo event.MatchRepository, tournamentRepo tournament.Repository) *AutoAssignTablesUseCase {
	return &AutoAssignTablesUseCase{
		matchRepo:      matchRepo,
		tournamentRepo: tournamentRepo,
	}
}

func (uc *AutoAssignTablesUseCase) Execute(ctx context.Context, tournamentID string) ([]event.Match, error) {
	t, err := uc.tournamentRepo.GetByIDDeep(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	occupiedTables := make(map[int]bool)
	var scheduledMatches []*event.Match
	var assignedMatches []event.Match

	// 1. Gather all occupied tables and scheduled matches across all events in the tournament
	for _, e := range t.Events {
		for i := range e.Matches {
			m := &e.Matches[i]
			if m.Status == "in_progress" && m.TableNumber != nil {
				occupiedTables[*m.TableNumber] = true
			} else if m.Status == "scheduled" && m.TableNumber == nil && len(m.TeamA) > 0 && len(m.TeamB) > 0 {
				scheduledMatches = append(scheduledMatches, m)
			}
		}
	}

	// 2. Determine available tables
	availableTables := make(map[int]bool)
	for i := 1; i <= t.NumTables; i++ {
		if !occupiedTables[i] {
			availableTables[i] = true
		}
	}

	if len(availableTables) == 0 || len(scheduledMatches) == 0 {
		return nil, nil
	}

	// 3. Assign tables based on priority
	for _, m := range scheduledMatches {
		if len(availableTables) == 0 {
			break
		}

		assignedTable := 0

		// Check division priority
		if m.DivisionID != "" {
			for _, pTable := range t.TablePriorityFor(m.DivisionID) {
				if availableTables[pTable] {
					assignedTable = pTable
					break
				}
			}
		}

		// Fallback to any available table
		if assignedTable == 0 {
			// Optional: We might only want to assign if they are not exclusively reserved,
			// but for now, any division can use any empty table if their priorities aren't met
			for tNum := range availableTables {
				assignedTable = tNum
				break
			}
		}

		if assignedTable > 0 {
			m.TableNumber = &assignedTable
			m.Status = "in_progress" // Start the match
			delete(availableTables, assignedTable)

			err := uc.matchRepo.Save(ctx, m)
			if err == nil {
				assignedMatches = append(assignedMatches, *m)
			}
		}
	}

	return assignedMatches, nil
}
