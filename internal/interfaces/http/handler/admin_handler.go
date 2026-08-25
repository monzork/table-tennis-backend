package handler

import (
	"sync"
	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/application/player"
	divisionDomain "table-tennis-backend/internal/domain/division"

	"github.com/gofiber/fiber/v2"
	eventUC "table-tennis-backend/internal/application/tournament"
)

type AdminHandler struct {
	playerUC       *player.RegisterPlayerUseCase
	tournamentUC   *event.CreateTournamentUseCase
	matchCreate    *match.CreateMatchUseCase
	matchList      *match.GetMatchesUseCase
	leaderboard    *leaderboard.GetLeaderboardUseCase
	getTournaments *event.GetTournamentsUseCase
	divisionUC     *division.DivisionUseCase
	eventGetAll    *eventUC.GetAllEventsUseCase
}

func NewAdminHandler(
	p *player.RegisterPlayerUseCase,
	t *event.CreateTournamentUseCase,
	mc *match.CreateMatchUseCase,
	ml *match.GetMatchesUseCase,
	l *leaderboard.GetLeaderboardUseCase,
	gt *event.GetTournamentsUseCase,
	duc *division.DivisionUseCase,
	ega *eventUC.GetAllEventsUseCase,
) *AdminHandler {
	return &AdminHandler{
		playerUC:       p,
		tournamentUC:   t,
		matchCreate:    mc,
		matchList:      ml,
		leaderboard:    l,
		getTournaments: gt,
		divisionUC:     duc,
		eventGetAll:    ega,
	}
}

func (h *AdminHandler) Dashboard(c *fiber.Ctx) error {
	type result struct {
		tournaments any
		events      any
		players     any
		divisions   any
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		e, _ := h.eventGetAll.Execute(c.Context())
		res.tournaments = e
	}()
	go func() {
		defer wg.Done()
		t, _ := h.getTournaments.Execute(c.Context())
		res.events = t
	}()
	go func() {
		defer wg.Done()
		p, _ := h.leaderboard.ExecuteSingles(c.Context())
		res.players = p
	}()
	go func() {
		defer wg.Done()
		d, _ := h.divisionUC.GetAll(c.Context())
		res.divisions = d
	}()
	wg.Wait()

	lang := getLang(c)
	return c.Render("admin/dashboard", merge(tMap(lang), fiber.Map{
		"Tournaments": res.tournaments,
		"Events":      res.events,
		"Players":     res.players,
		"Divisions":   res.divisions,
	}), "layouts/admin")
}

func (h *AdminHandler) Players(c *fiber.Ctx) error {
	// For now, get players from leaderboard since it lists them
	board, err := h.leaderboard.ExecuteSingles(c.Context())
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	events, err := h.getTournaments.Execute(c.Context())
	var activeTournaments []any
	if err == nil {
		for _, t := range events {
			if t.Status != "finished" {
				activeTournaments = append(activeTournaments, t)
			}
		}
	}
	lang := getLang(c)
	return c.Render("admin/players", merge(tMap(lang), fiber.Map{
		"Players": board,
		"Events":  activeTournaments,
	}), "layouts/admin")
}

func (h *AdminHandler) Tournaments(c *fiber.Ctx) error {
	type result struct {
		tournaments any
		divisions   any
		players     any
		standalone  any
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		e, _ := h.eventGetAll.Execute(c.Context())
		res.tournaments = e
	}()
	go func() {
		defer wg.Done()
		d, _ := h.divisionUC.GetAll(c.Context())
		res.divisions = d
	}()
	go func() {
		defer wg.Done()
		p, _ := h.leaderboard.ExecuteSingles(c.Context())
		res.players = p
	}()
	go func() {
		defer wg.Done()
		t, _ := h.getTournaments.Execute(c.Context())
		var sa []any
		// We use type assertion since it returns []*tournamentDomain.Event
		for _, tourney := range t {
			if tourney.TournamentID == nil || *tourney.TournamentID == "" {
				sa = append(sa, tourney)
			}
		}
		res.standalone = sa
	}()
	wg.Wait()

	lang := getLang(c)
	return c.Render("admin/tournaments", merge(tMap(lang), fiber.Map{
		"Tournaments": res.tournaments,
		"Divisions":   res.divisions,
		"Players":     res.players,
		"Standalone":  res.standalone,
	}), "layouts/admin")
}
func (h *AdminHandler) Divisions(c *fiber.Ctx) error {
	divisions, err := h.divisionUC.GetAll(c.Context())
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	lang := getLang(c)
	return c.Render("admin/divisions", merge(tMap(lang), fiber.Map{
		"Divisions": divisions,
	}), "layouts/admin")
}

// NewPlayerField returns an empty player field row for inline new-player entry in event creation.
func (h *AdminHandler) NewPlayerField(c *fiber.Ctx) error {
	lang := getLang(c)
	return c.Render("admin/partials/new-player-field", tMap(lang))
}

// CloseModal returns an empty response so HTMX can clear the modal root container.
func (h *AdminHandler) CloseModal(c *fiber.Ctx) error {
	return c.SendString("")
}

func (h *AdminHandler) DivisionSelect(c *fiber.Ctx) error {
	skipElo := c.Query("skipElo") == "true"
	if skipElo {
		return c.SendString("")
	}
	divisions, err := h.divisionUC.GetAll(c.Context())
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	// New tournaments can only be created against gender-specific division
	// bands (or no division at all, via the "none" option hardcoded in the
	// template) -- the old gender-agnostic bands (Champion/First/Second/
	// Third/Fourth) are retired from selection for new tournaments, but are
	// intentionally NOT deleted from the divisions table: bracket rendering
	// for every existing tournament still built against them looks them up
	// by name (see bracket.BuildBracket), so removing the rows would break
	// those tournaments' displays.
	selectable := divisionDomain.OnlyGendered(divisions)

	lang := getLang(c)
	return c.Render("admin/partials/division-select-options", merge(tMap(lang), fiber.Map{
		"Divisions": selectable,
	}))
}
