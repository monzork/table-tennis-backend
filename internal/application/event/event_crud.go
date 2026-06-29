package event

import (
	"context"
	"fmt"
	divisionDomain "table-tennis-backend/internal/domain/division"
	eventDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	"time"
)

type CategoryConfig struct {
	Auto           bool
	Format         string
	GroupPassCount int
	PlayerIDs      []string
}

type CreateEventUseCase struct {
	eventRepo      eventDomain.Repository
	tournamentRepo tournamentDomain.Repository
	playerRepo     playerDomain.Repository
	divisionRepo   divisionDomain.Repository
}

func NewCreateEventUseCase(
	eventRepo eventDomain.Repository,
	tournamentRepo tournamentDomain.Repository,
	playerRepo playerDomain.Repository,
	divisionRepo divisionDomain.Repository,
) *CreateEventUseCase {
	return &CreateEventUseCase{
		eventRepo:      eventRepo,
		tournamentRepo: tournamentRepo,
		playerRepo:     playerRepo,
		divisionRepo:   divisionRepo,
	}
}

func (uc *CreateEventUseCase) Execute(
	ctx context.Context,
	name string,
	divisionIDs []string,
	skipElo bool,
	startDateStr, endDateStr string,
	singlesMen, singlesWomen, doublesMen, doublesWomen, doublesMixed, teamsMen, teamsWomen CategoryConfig,
	existingTournamentIDs []string,
) (*eventDomain.Event, error) {
	start, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return nil, err
	}

	e, err := eventDomain.NewEvent(idgen.Generate(), name, divisionIDs, skipElo, start, end)
	if err != nil {
		return nil, err
	}

	var divs []*divisionDomain.Division
	if !skipElo && len(divisionIDs) > 0 {
		for _, did := range divisionIDs {
			if did != "" && did != "none" {
				div, err := uc.divisionRepo.GetById(ctx, did)
				if err != nil {
					return nil, fmt.Errorf("failed to fetch division: %w", err)
				}
				divs = append(divs, div)
			}
		}
	}

	// Collect all unique player IDs across all categories and batch-load them
	allIDSet := make(map[string]bool)
	for _, cfg := range []CategoryConfig{singlesMen, singlesWomen, doublesMen, doublesWomen, doublesMixed, teamsMen, teamsWomen} {
		for _, idStr := range cfg.PlayerIDs {
			if idStr != "" {
				allIDSet[idStr] = true
			}
		}
	}
	allIDs := make([]string, 0, len(allIDSet))
	for id := range allIDSet {
		allIDs = append(allIDs, id)
	}
	playerCache := make(map[string]*playerDomain.Player)
	if len(allIDs) > 0 {
		loaded, err := uc.playerRepo.GetByIDs(ctx, allIDs)
		if err == nil {
			for _, p := range loaded {
				playerCache[p.ID] = p
			}
		}
	}

	// Helper to create a tournament under this event
	createSubTourney := func(tName string, tType string, tFormat string, category string, groupPassCount int, players []*playerDomain.Player) error {
		t, err := tournamentDomain.NewTournament(idgen.Generate(), tName, tType, tFormat, category, start, end, []tournamentDomain.Rule{}, groupPassCount, players, false)
		if err != nil {
			return err
		}
		t.EventID = &e.ID
		t.SkipElo = skipElo
		e.Tournaments = append(e.Tournaments, t)
		return nil
	}

	// Helper to get qualified players for a category (from cache)
	getPlayers := func(ids []string, gender string, isDoubles bool) []*playerDomain.Player {
		var players []*playerDomain.Player
		for _, idStr := range ids {
			p, ok := playerCache[idStr]
			if !ok {
				continue
			}
			if gender != "" && p.Gender != gender {
				continue
			}
			if !skipElo && len(divs) > 0 {
				eloVal := p.SinglesElo
				if isDoubles {
					eloVal = p.DoublesElo
				}
				matched := false
				for _, div := range divs {
					if div.ContainsElo(int16(eloVal)) {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}
			players = append(players, p)
		}
		return players
	}

	processCategory := func(cfg CategoryConfig, suffix, tType, categoryGender string, isDoubles bool) {
		if !cfg.Auto {
			return
		}
		
		allCatPlayers := getPlayers(cfg.PlayerIDs, categoryGender, isDoubles)
		if len(allCatPlayers) == 0 {
			return
		}

		if skipElo || len(divs) == 0 {
			tName := fmt.Sprintf("%s - %s", e.Name, suffix)
			if categoryGender == "men" || categoryGender == "women" {
			   // fallback in case I used lowercase men/women instead of M/F, wait, categoryGender is M/F, but for category we pass men/women, wait!
			}
			catArg := categoryGender
			if categoryGender == "M" { catArg = "men" } else if categoryGender == "F" { catArg = "women" } else { catArg = "open" }
			_ = createSubTourney(tName, tType, cfg.Format, catArg, cfg.GroupPassCount, allCatPlayers)
		} else {
			// Group by division
			for _, div := range divs {
				var divPlayers []*playerDomain.Player
				for _, p := range allCatPlayers {
					eloVal := p.SinglesElo
					if isDoubles {
						eloVal = p.DoublesElo
					}
					if div.ContainsElo(int16(eloVal)) {
						divPlayers = append(divPlayers, p)
					}
				}
				
				if len(divPlayers) > 0 {
					tName := fmt.Sprintf("%s - %s (%s)", e.Name, suffix, div.Name)
					catArg := categoryGender
					if categoryGender == "M" { catArg = "men" } else if categoryGender == "F" { catArg = "women" } else { catArg = "open" }
					_ = createSubTourney(tName, tType, cfg.Format, catArg, cfg.GroupPassCount, divPlayers)
				}
			}
		}
	}

	processCategory(singlesMen, "Men's Singles", "singles", "M", false)
	processCategory(singlesWomen, "Women's Singles", "singles", "F", false)
	processCategory(doublesMen, "Men's Doubles", "doubles", "M", true)
	processCategory(doublesWomen, "Women's Doubles", "doubles", "F", true)
	processCategory(doublesMixed, "Mixed Doubles", "doubles", "", true)
	processCategory(teamsMen, "Men's Teams", "teams", "M", false)
	processCategory(teamsWomen, "Women's Teams", "teams", "F", false)



	if err := uc.eventRepo.Save(ctx, e); err != nil {
		return nil, err
	}

	var validTournamentIDs []string
	for _, tID := range existingTournamentIDs {
		if tID != "" {
			validTournamentIDs = append(validTournamentIDs, tID)
		}
	}
	if len(validTournamentIDs) > 0 {
		_ = uc.tournamentRepo.UpdateEventIDBulk(ctx, validTournamentIDs, e.ID)
	}

	// Reload the event with loaded tournaments
	return uc.eventRepo.GetByID(ctx, e.ID)
}

type GetEventByIDUseCase struct {
	eventRepo eventDomain.Repository
}

func NewGetEventByIDUseCase(eventRepo eventDomain.Repository) *GetEventByIDUseCase {
	return &GetEventByIDUseCase{eventRepo: eventRepo}
}

func (uc *GetEventByIDUseCase) Execute(ctx context.Context, idStr string) (*eventDomain.Event, error) {
	return uc.eventRepo.GetByID(ctx, idStr)
}

type GetAllEventsUseCase struct {
	eventRepo eventDomain.Repository
}

func NewGetAllEventsUseCase(eventRepo eventDomain.Repository) *GetAllEventsUseCase {
	return &GetAllEventsUseCase{eventRepo: eventRepo}
}

func (uc *GetAllEventsUseCase) Execute(ctx context.Context) ([]*eventDomain.Event, error) {
	return uc.eventRepo.GetAll(ctx)
}

type DeleteEventUseCase struct {
	eventRepo eventDomain.Repository
}

func NewDeleteEventUseCase(eventRepo eventDomain.Repository) *DeleteEventUseCase {
	return &DeleteEventUseCase{eventRepo: eventRepo}
}

func (uc *DeleteEventUseCase) Execute(ctx context.Context, idStr string) error {
	return uc.eventRepo.Delete(ctx, idStr)
}

func (uc *DeleteEventUseCase) ExecuteBulk(ctx context.Context, idStrs []string) error {
	return uc.eventRepo.DeleteEvents(ctx, idStrs)
}
