package handler

import (
	"context"
	"table-tennis-backend/internal/application/tournament"

	"github.com/gofiber/fiber/v2"
)

type TournamentHandler struct {
	createUC *tournament.CreateTournamentUseCase
}

func NewTournamentHandler(createUC *tournament.CreateTournamentUseCase) *TournamentHandler {
	return &TournamentHandler{createUC: createUC}
}

func (h *TournamentHandler) Create(c *fiber.Ctx) error {
	var body struct {
		Name      string `json:"name" form:"name"`
		Type      string `json:"type" form:"type"`
		StartDate string `json:"startDate" form:"startDate"`
		EndDate   string `json:"endDate" form:"endDate"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	t, err := h.createUC.Execute(context.Background(), body.Name, body.Type, body.StartDate, body.EndDate)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("tournament-row", t)
}
