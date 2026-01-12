package handler

import (
	"context"
	"table-tennis-backend/internal/application/player"

	"github.com/gofiber/fiber/v2"
)

type PlayerHandler struct {
	registerPlayerUC *player.RegisterPlayerUseCase
}

func NewPlayerHandler(uc *player.RegisterPlayerUseCase) *PlayerHandler {
	return &PlayerHandler{registerPlayerUC: uc}
}

func (h *PlayerHandler) Register(c *fiber.Ctx) error {
	var body struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Birthdate string `json:"birthdate"`
		Country   string `json:"country"`
	}

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	player, err := h.registerPlayerUC.Execute(context.Background(), body.FirstName, body.LastName, body.Birthdate, body.Country)

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("player-row", player)
}
