package handler

import (
	"table-tennis-backend/internal/domain/tournament"
	bunPlayer "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/gofiber/fiber/v2"
)

type DashboardHandler struct{}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

func (h *DashboardHandler) Show(c *fiber.Ctx) error {
	// Load all players
	var players []*bunPlayer.PlayerModel
	if err := bunPlayer.DB.NewSelect().Model(&players).Scan(c.Context()); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Load all tournaments
	var tournaments []*bunPlayer.TournamentModel
	if err := bunPlayer.DB.NewSelect().Model(&tournaments).Scan(c.Context()); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Load matches (simplified)
	var matches []*tournament.Match
	// if err := bunPlayer.DB.NewSelect().Model(&matches).Scan(c.Context()); err != nil {
	// 	return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	// }

	return c.Render("base", fiber.Map{
		"Players":     players,
		"Tournaments": tournaments,
		"Matches":     matches,
	})
}
