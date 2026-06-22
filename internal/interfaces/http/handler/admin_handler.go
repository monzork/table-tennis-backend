package handler

import (
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/application/tournament"
	"table-tennis-backend/internal/application/division"

	"github.com/gofiber/fiber/v2"
	eventUC "table-tennis-backend/internal/application/event"
)

type AdminHandler struct {
	playerUC     *player.RegisterPlayerUseCase
	tournamentUC *tournament.CreateTournamentUseCase
	matchCreate  *match.CreateMatchUseCase
	matchList      *match.GetMatchesUseCase
	leaderboard    *leaderboard.GetLeaderboardUseCase
	getTournaments *tournament.GetTournamentsUseCase
	divisionUC     *division.DivisionUseCase
	eventGetAll    *eventUC.GetAllEventsUseCase
}

func NewAdminHandler(
	p *player.RegisterPlayerUseCase,
	t *tournament.CreateTournamentUseCase,
	mc *match.CreateMatchUseCase,
	ml *match.GetMatchesUseCase,
	l *leaderboard.GetLeaderboardUseCase,
	gt *tournament.GetTournamentsUseCase,
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
	events, _ := h.eventGetAll.Execute(c.Context())
	tournaments, _ := h.getTournaments.Execute(c.Context())
	players, _ := h.leaderboard.ExecuteSingles(c.Context())
	divisions, _ := h.divisionUC.GetAll(c.Context())

	return c.Render("admin/dashboard", fiber.Map{
		"Events":      events,
		"Tournaments": tournaments,
		"Players":     players,
		"Divisions":   divisions,
	}, "layouts/admin")
}

func (h *AdminHandler) Players(c *fiber.Ctx) error {
	// For now, get players from leaderboard since it lists them
	board, err := h.leaderboard.ExecuteSingles(c.Context())
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.Render("admin/players", fiber.Map{
		"Players": board,
	}, "layouts/admin")
}

func (h *AdminHandler) Tournaments(c *fiber.Ctx) error {
	players, _ := h.leaderboard.ExecuteSingles(c.Context())
	tourneys, _ := h.getTournaments.Execute(c.Context())
	divisions, _ := h.divisionUC.GetAll(c.Context())
	return c.Render("admin/tournaments", fiber.Map{
		"Players":     players,
		"Tournaments": tourneys,
		"Divisions":   divisions,
	}, "layouts/admin")
}

func (h *AdminHandler) Events(c *fiber.Ctx) error {
	events, _ := h.eventGetAll.Execute(c.Context())
	divisions, _ := h.divisionUC.GetAll(c.Context())
	players, _ := h.leaderboard.ExecuteSingles(c.Context())
	return c.Render("admin/events", fiber.Map{
		"Events":    events,
		"Divisions": divisions,
		"Players":   players,
	}, "layouts/admin")
}
func (h *AdminHandler) Divisions(c *fiber.Ctx) error {
	divisions, err := h.divisionUC.GetAll(c.Context())
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.Render("admin/divisions", fiber.Map{
		"Divisions": divisions,
	}, "layouts/admin")
}

// NewPlayerField returns an empty player field row for inline new-player entry in tournament creation.
func (h *AdminHandler) NewPlayerField(c *fiber.Ctx) error {
	return c.Render("admin/partials/new-player-field", nil)
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
	return c.Render("admin/partials/division-select-options", fiber.Map{
		"Divisions": divisions,
	})
}
