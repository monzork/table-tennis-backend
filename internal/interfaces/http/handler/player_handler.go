package handler

import (
	"context"
	"table-tennis-backend/internal/application/player"

	"github.com/gofiber/fiber/v2"
)

type PlayerHandler struct {
	registerPlayerUC *player.RegisterPlayerUseCase
	updatePlayerUC   *player.UpdatePlayerUseCase
	deletePlayerUC   *player.DeletePlayerUseCase
}

func NewPlayerHandler(
	uc *player.RegisterPlayerUseCase,
	uuc *player.UpdatePlayerUseCase,
	duc *player.DeletePlayerUseCase,
) *PlayerHandler {
	return &PlayerHandler{
		registerPlayerUC: uc,
		updatePlayerUC:   uuc,
		deletePlayerUC:   duc,
	}
}

func (h *PlayerHandler) Register(c *fiber.Ctx) error {
	var body struct {
		FirstName string `json:"firstName" form:"firstName"`
		LastName  string `json:"lastName" form:"lastName"`
		Birthdate string `json:"birthdate" form:"birthdate"`
		Country   string `json:"country" form:"country"`
		Gender    string `json:"gender" form:"gender"`
	}

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	player, err := h.registerPlayerUC.Execute(context.Background(), body.FirstName, body.LastName, body.Birthdate, body.Gender, body.Country)

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("admin/partials/player-row", player)
}

func (h *PlayerHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		FirstName string `json:"firstName" form:"firstName"`
		LastName  string `json:"lastName" form:"lastName"`
		Birthdate string `json:"birthdate" form:"birthdate"`
		Country   string `json:"country" form:"country"`
		Gender    string `json:"gender" form:"gender"`
	}

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	player, err := h.updatePlayerUC.Execute(c.Context(), id, body.FirstName, body.LastName, body.Birthdate, body.Gender, body.Country)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("admin/partials/player-row", player)
}

func (h *PlayerHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.deletePlayerUC.Execute(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusOK)
}
