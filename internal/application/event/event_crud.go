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
	divisionID string,
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

	e, err := eventDomain.NewEvent(idgen.Generate(), name, divisionID, skipElo, start, end)
	if err != nil {
		return nil, err
	}

	var div *divisionDomain.Division
	if !skipElo && divisionID != "" {
		var err error
		div, err = uc.divisionRepo.GetById(ctx, divisionID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch division: %w", err)
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
			if !skipElo && div != nil {
				eloVal := p.SinglesElo
				if isDoubles {
					eloVal = p.DoublesElo
				}
				if !div.ContainsElo(int16(eloVal)) {
					continue
				}
			}
			players = append(players, p)
		}
		return players
	}

	getNameWithDiv := func(suffix string) string {
		if div != nil {
			return fmt.Sprintf("%s - %s (%s)", e.Name, suffix, div.Name)
		}
		return fmt.Sprintf("%s - %s", e.Name, suffix)
	}

	if singlesMen.Auto {
		players := getPlayers(singlesMen.PlayerIDs, "M", false)
		tName := getNameWithDiv("Men's Singles")
		_ = createSubTourney(tName, "singles", singlesMen.Format, "men", singlesMen.GroupPassCount, players)
	}

	if singlesWomen.Auto {
		players := getPlayers(singlesWomen.PlayerIDs, "F", false)
		tName := getNameWithDiv("Women's Singles")
		_ = createSubTourney(tName, "singles", singlesWomen.Format, "women", singlesWomen.GroupPassCount, players)
	}

	if doublesMen.Auto {
		players := getPlayers(doublesMen.PlayerIDs, "M", true)
		tName := getNameWithDiv("Men's Doubles")
		_ = createSubTourney(tName, "doubles", doublesMen.Format, "men", doublesMen.GroupPassCount, players)
	}

	if doublesWomen.Auto {
		players := getPlayers(doublesWomen.PlayerIDs, "F", true)
		tName := getNameWithDiv("Women's Doubles")
		_ = createSubTourney(tName, "doubles", doublesWomen.Format, "women", doublesWomen.GroupPassCount, players)
	}

	if doublesMixed.Auto {
		players := getPlayers(doublesMixed.PlayerIDs, "", true)
		tName := getNameWithDiv("Mixed Doubles")
		_ = createSubTourney(tName, "doubles", doublesMixed.Format, "open", doublesMixed.GroupPassCount, players)
	}

	if teamsMen.Auto {
		players := getPlayers(teamsMen.PlayerIDs, "M", false)
		tName := getNameWithDiv("Men's Teams")
		_ = createSubTourney(tName, "teams", teamsMen.Format, "men", teamsMen.GroupPassCount, players)
	}

	if teamsWomen.Auto {
		players := getPlayers(teamsWomen.PlayerIDs, "F", false)
		tName := getNameWithDiv("Women's Teams")
		_ = createSubTourney(tName, "teams", teamsWomen.Format, "women", teamsWomen.GroupPassCount, players)
	}

	if err := uc.eventRepo.Save(ctx, e); err != nil {
		return nil, err
	}

	for _, tID := range existingTournamentIDs {
		if tID == "" {
			continue
		}
		t, err := uc.tournamentRepo.GetByID(ctx, tID)
		if err == nil {
			t.EventID = &e.ID
			_ = uc.tournamentRepo.Update(ctx, t)
		}
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
