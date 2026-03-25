package handler

import (
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/application/tournament"

	"github.com/gofiber/fiber/v2"
)

type AdminHandler struct {
	playerUC     *player.RegisterPlayerUseCase
	tournamentUC *tournament.CreateTournamentUseCase
	matchCreate  *match.CreateMatchUseCase
	matchList      *match.GetMatchesUseCase
	leaderboard    *leaderboard.GetLeaderboardUseCase
	getTournaments *tournament.GetTournamentsUseCase
}

func NewAdminHandler(
	p *player.RegisterPlayerUseCase,
	t *tournament.CreateTournamentUseCase,
	mc *match.CreateMatchUseCase,
	ml *match.GetMatchesUseCase,
	l *leaderboard.GetLeaderboardUseCase,
	gt *tournament.GetTournamentsUseCase,
) *AdminHandler {
	return &AdminHandler{
		playerUC:       p,
		tournamentUC:   t,
		matchCreate:    mc,
		matchList:      ml,
		leaderboard:    l,
		getTournaments: gt,
	}
}

func (h *AdminHandler) Dashboard(c *fiber.Ctx) error {
	// Let's just render the dashboard template
	return c.Render("admin/dashboard", fiber.Map{}, "layouts/admin")
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
	return c.Render("admin/tournaments", fiber.Map{
		"Players":     players,
		"Tournaments": tourneys,
	}, "layouts/admin")
}

func (h *AdminHandler) Matches(c *fiber.Ctx) error {
	matches, err := h.matchList.GetAllViews(c.Context())
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	// We also need to send players and tournaments to populate the dropdowns
	players, _ := h.leaderboard.ExecuteSingles(c.Context())
	
	return c.Render("admin/matches", fiber.Map{
		"Matches": matches,
		"Players": players,
	}, "layouts/admin")
}
