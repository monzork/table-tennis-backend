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

func (h *LeaderboardHandler) GetSingles(c *fiber.Ctx) error {
	players, err := h.getUC.Execute(c.Context(), "singles")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("rankings", fiber.Map{
		"Players": players,
		"Type":    "Singles",
	}, "layouts/public")
}

func (h *LeaderboardHandler) GetDoubles(c *fiber.Ctx) error {
	players, err := h.getUC.Execute(c.Context(), "doubles")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("rankings", fiber.Map{
		"Players": players,
		"Type":    "Doubles",
	}, "layouts/public")
}
