package event

import (
	"context"
	"fmt"
	divisionDomain "table-tennis-backend/internal/domain/division"
	eventDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	"time"

	"github.com/google/uuid"
)

type CategoryConfig struct {
	Auto           bool
	Format         string
	GroupPassCount int
	PlayerIDs      []string
}

type CreateEventUseCase struct {
	eventRepo      *bun.EventRepository
	tournamentRepo *bun.TournamentRepository
	playerRepo     *bun.PlayerRepository
	divisionRepo   *bun.DivisionRepository
}

func NewCreateEventUseCase(
	eventRepo *bun.EventRepository,
	tournamentRepo *bun.TournamentRepository,
	playerRepo *bun.PlayerRepository,
	divisionRepo *bun.DivisionRepository,
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
) (*eventDomain.Event, error) {
	start, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return nil, err
	}

	e, err := eventDomain.NewEvent(name, divisionID, skipElo, start, end)
	if err != nil {
		return nil, err
	}

	tx, err := uc.eventRepo.DB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.NewInsert().Model(&bun.EventModel{
		ID:         e.ID,
		Name:       e.Name,
		DivisionID: e.DivisionID,
		SkipElo:    e.SkipElo,
		StartDate:  e.StartDate,
		EndDate:    e.EndDate,
	}).Exec(ctx); err != nil {
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
	allIDSet := make(map[uuid.UUID]bool)
	for _, cfg := range []CategoryConfig{singlesMen, singlesWomen, doublesMen, doublesWomen, doublesMixed, teamsMen, teamsWomen} {
		for _, idStr := range cfg.PlayerIDs {
			if id, err := uuid.Parse(idStr); err == nil {
				allIDSet[id] = true
			}
		}
	}
	allIDs := make([]uuid.UUID, 0, len(allIDSet))
	for id := range allIDSet {
		allIDs = append(allIDs, id)
	}
	playerCache := make(map[uuid.UUID]*playerDomain.Player)
	if len(allIDs) > 0 {
		loaded, err := uc.playerRepo.GetByIDs(ctx, allIDs)
		if err == nil {
			for _, p := range loaded {
				playerCache[p.ID] = p
			}
		}
	}

	// Helper to create a tournament under this event (within the shared transaction)
	createSubTourney := func(tName string, tType string, tFormat string, category string, groupPassCount int, players []*playerDomain.Player) error {
		t, err := tournamentDomain.NewTournament(tName, tType, tFormat, category, start, end, []tournamentDomain.Rule{}, groupPassCount, players)
		if err != nil {
			return err
		}
		t.EventID = &e.ID
		t.SkipElo = skipElo
		return uc.tournamentRepo.SaveTx(ctx, tx, t)
	}

	// Helper to get qualified players for a category (from cache)
	getPlayers := func(ids []string, gender string, isDoubles bool) []*playerDomain.Player {
		var players []*playerDomain.Player
		for _, idStr := range ids {
			id, err := uuid.Parse(idStr)
			if err != nil {
				continue
			}
			p, ok := playerCache[id]
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

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Reload the event with loaded tournaments
	return uc.eventRepo.GetByID(ctx, e.ID)
}

type GetEventByIDUseCase struct {
	eventRepo *bun.EventRepository
}

func NewGetEventByIDUseCase(eventRepo *bun.EventRepository) *GetEventByIDUseCase {
	return &GetEventByIDUseCase{eventRepo: eventRepo}
}

func (uc *GetEventByIDUseCase) Execute(ctx context.Context, idStr string) (*eventDomain.Event, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}
	return uc.eventRepo.GetByID(ctx, id)
}

type GetAllEventsUseCase struct {
	eventRepo *bun.EventRepository
}

func NewGetAllEventsUseCase(eventRepo *bun.EventRepository) *GetAllEventsUseCase {
	return &GetAllEventsUseCase{eventRepo: eventRepo}
}

func (uc *GetAllEventsUseCase) Execute(ctx context.Context) ([]*eventDomain.Event, error) {
	return uc.eventRepo.GetAll(ctx)
}

type DeleteEventUseCase struct {
	eventRepo *bun.EventRepository
}

func NewDeleteEventUseCase(eventRepo *bun.EventRepository) *DeleteEventUseCase {
	return &DeleteEventUseCase{eventRepo: eventRepo}
}

func (uc *DeleteEventUseCase) Execute(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	return uc.eventRepo.Delete(ctx, id)
}
