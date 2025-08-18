package handler

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/monzork/table-tennis-backend/internal/domain/players"
)

type PlayersHandler struct {
	service *players.Service
}

func NewPlayersHandler(service *players.Service) *PlayersHandler {
	return &PlayersHandler{service: service}
}

func (h *PlayersHandler) Register(c fiber.Ctx) error {
	var body struct {
		Name      string `json:"name"`
		Sex       string `json:"sex"`
		Country   string `json:"country"`
		City      string `json:"city"`
		Birthdate string `json:"birthdate"`
		Elo       *int16 `json:"elo,omitempty"`
	}

	if err := c.Bind().Body(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	player, err := h.service.RegisterPlayers(
		context.Background(),
		body.Name,
		body.Sex,
		body.Country,
		body.City,
		body.Birthdate,
		body.Elo,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(player)
}
