package handler

import (
	"table-tennis-backend/internal/application/leaderboard"

	"github.com/gofiber/fiber/v2"
)

type LeaderboardHandler struct {
	getUC *leaderboard.GetLeaderboardUseCase
}

func NewLeaderboardHandler(uc *leaderboard.GetLeaderboardUseCase) *LeaderboardHandler {
	return &LeaderboardHandler{getUC: uc}
}

// Returns leaderboard partial (for HTMX)
func (h *LeaderboardHandler) Get(c *fiber.Ctx) error {
	players, err := h.getUC.Execute(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("partials/leaderboard.html", fiber.Map{
		"Players": players,
	})
}
