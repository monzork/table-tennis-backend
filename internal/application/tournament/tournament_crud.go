package tournament

import (
	"context"
	"fmt"
	"strings"
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

// CustomEventConfig defines one manually-named child event with a hand-picked
// player roster, bypassing gender/division auto-grouping entirely.
type CustomEventConfig struct {
	Name           string
	Format         string
	GroupPassCount int
	PlayerIDs      []string
}

type CreateEventUseCase struct {
	tournamentRepo tournamentDomain.Repository
	eventRepo      eventDomain.Repository
	playerRepo     playerDomain.Repository
	divisionRepo   divisionDomain.Repository
}

func NewCreateEventUseCase(
	tournamentRepo tournamentDomain.Repository,
	eventRepo eventDomain.Repository,
	playerRepo playerDomain.Repository,
	divisionRepo divisionDomain.Repository,
) *CreateEventUseCase {
	return &CreateEventUseCase{
		tournamentRepo: tournamentRepo,
		eventRepo:      eventRepo,
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
	singlesMen, singlesWomen, doublesMen, doublesWomen, doublesMixed, teamsMen, teamsWomen, singlesOpen CategoryConfig,
	customSinglesEvents []CustomEventConfig,
	existingTournamentIDs []string,
) (*tournamentDomain.Tournament, error) {
	start, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return nil, err
	}

	e, err := tournamentDomain.NewTournament(idgen.Generate(), name, divisionIDs, skipElo, start, end)
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
	for _, cfg := range []CategoryConfig{singlesMen, singlesWomen, doublesMen, doublesWomen, doublesMixed, teamsMen, teamsWomen, singlesOpen} {
		if !cfg.Auto {
			continue
		}
		for _, idStr := range cfg.PlayerIDs {
			if idStr != "" {
				allIDSet[idStr] = true
			}
		}
	}
	for _, cfg := range customSinglesEvents {
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

	// Helper to create a event under this tournament
	createSubTourney := func(tName string, tType string, tFormat string, category string, groupPassCount int, players []*playerDomain.Player, skipDivisionSplit bool, useGenderDivisions bool) error {
		t, err := eventDomain.NewEvent(idgen.Generate(), tName, tType, tFormat, category, start, end, []eventDomain.Rule{}, groupPassCount, players, false)
		if err != nil {
			return err
		}
		t.TournamentID = &e.ID
		t.SkipElo = skipElo
		t.NumTables = e.NumTables
		t.SkipDivisionSplit = skipDivisionSplit
		t.UseGenderDivisions = useGenderDivisions
		e.Events = append(e.Events, t)
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
			if categoryGender == "M" {
				catArg = "men"
			} else if categoryGender == "F" {
				catArg = "women"
			} else {
				catArg = "open"
			}
			_ = createSubTourney(tName, tType, cfg.Format, catArg, cfg.GroupPassCount, allCatPlayers, len(divs) == 0, false)
		} else {
			// Group by division
			for _, div := range divs {
				// A division only applies to categories of its own gender --
				// Gender=="both" divisions apply to every category, but a
				// gender-specific division (e.g. a new "1st Division (Men)"
				// band) must not also pull players into a differently-
				// gendered category just because their Elo happens to fall
				// in that band's numeric range too.
				if !div.MatchesGender(categoryGender) {
					continue
				}

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
					if categoryGender == "M" {
						catArg = "men"
					} else if categoryGender == "F" {
						catArg = "women"
					} else {
						catArg = "open"
					}
					useGenderDivisions := div.Gender != "" && !strings.EqualFold(div.Gender, "both")
					_ = createSubTourney(tName, tType, cfg.Format, catArg, cfg.GroupPassCount, divPlayers, false, useGenderDivisions)
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
	processCategory(singlesOpen, "Open Singles", "singles", "", false)

	for _, cfg := range customSinglesEvents {
		var players []*playerDomain.Player
		for _, idStr := range cfg.PlayerIDs {
			if p, ok := playerCache[idStr]; ok {
				players = append(players, p)
			}
		}
		if len(players) == 0 || cfg.Name == "" {
			continue
		}
		tName := fmt.Sprintf("%s - %s", e.Name, cfg.Name)
		_ = createSubTourney(tName, "singles", cfg.Format, "open", cfg.GroupPassCount, players, true, false)
	}

	if err := uc.tournamentRepo.Save(ctx, e); err != nil {
		return nil, err
	}

	var validTournamentIDs []string
	for _, tID := range existingTournamentIDs {
		if tID != "" {
			validTournamentIDs = append(validTournamentIDs, tID)
		}
	}
	if len(validTournamentIDs) > 0 {
		_ = uc.eventRepo.UpdateEventIDBulk(ctx, validTournamentIDs, e.ID)
	}

	// Reload the tournament with loaded events
	return uc.tournamentRepo.GetByID(ctx, e.ID)
}

type GetEventByIDUseCase struct {
	tournamentRepo tournamentDomain.Repository
}

func NewGetEventByIDUseCase(tournamentRepo tournamentDomain.Repository) *GetEventByIDUseCase {
	return &GetEventByIDUseCase{tournamentRepo: tournamentRepo}
}

func (uc *GetEventByIDUseCase) Execute(ctx context.Context, idStr string) (*tournamentDomain.Tournament, error) {
	return uc.tournamentRepo.GetByIDDeep(ctx, idStr)
}

type GetAllEventsUseCase struct {
	tournamentRepo tournamentDomain.Repository
}

func NewGetAllEventsUseCase(tournamentRepo tournamentDomain.Repository) *GetAllEventsUseCase {
	return &GetAllEventsUseCase{tournamentRepo: tournamentRepo}
}

func (uc *GetAllEventsUseCase) Execute(ctx context.Context) ([]*tournamentDomain.Tournament, error) {
	return uc.tournamentRepo.GetAll(ctx)
}

type DeleteEventUseCase struct {
	tournamentRepo tournamentDomain.Repository
}

func NewDeleteEventUseCase(tournamentRepo tournamentDomain.Repository) *DeleteEventUseCase {
	return &DeleteEventUseCase{tournamentRepo: tournamentRepo}
}

func (uc *DeleteEventUseCase) Execute(ctx context.Context, idStr string) error {
	return uc.tournamentRepo.Delete(ctx, idStr)
}

func (uc *DeleteEventUseCase) ExecuteBulk(ctx context.Context, idStrs []string) error {
	return uc.tournamentRepo.DeleteEvents(ctx, idStrs)
}

// ── Update ──────────────────────────────────────────────────────────────────

type UpdateEventUseCase struct {
	tournamentRepo tournamentDomain.Repository
}

func NewUpdateEventUseCase(tournamentRepo tournamentDomain.Repository) *UpdateEventUseCase {
	return &UpdateEventUseCase{tournamentRepo: tournamentRepo}
}

func (uc *UpdateEventUseCase) Execute(ctx context.Context, idStr, name, startDateStr, endDateStr string, numTables int, tablePriorities map[string][]int, federationEndorsed bool) (*tournamentDomain.Tournament, error) {
	e, err := uc.tournamentRepo.GetByID(ctx, idStr)
	if err != nil {
		return nil, err
	}
	if name != "" {
		e.Name = name
	}
	if startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			e.StartDate = t
		}
	}
	if endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			e.EndDate = t
		}
	}
	if numTables > 0 {
		e.NumTables = numTables
	}
	if tablePriorities != nil {
		e.TablePriorities = tablePriorities
	}
	e.FederationEndorsed = federationEndorsed
	if err := uc.tournamentRepo.Update(ctx, e); err != nil {
		return nil, fmt.Errorf("failed to update tournament: %w", err)
	}
	return e, nil
}
