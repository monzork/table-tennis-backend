package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/monzork/table-tennis-backend/internal/domain/players"
)

type PlayersHandler struct {
	service *players.Service
}

func NewPlayersHandler(service *players.Service) *PlayersHandler {
	return &PlayersHandler{service: service}
}

func (h *PlayersHandler) RegisterPlayers(c fiber.Ctx) error {
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

	return c.Render("players-row", player)
}

func (h *PlayersHandler) GetAllPlayers(c fiber.Ctx) error {
	playersList, err := h.service.GetAllPlayers(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("partials/players-list", fiber.Map{
		"Players": playersList,
	})
}

func (h *PlayersHandler) ShowPlayersTab(c fiber.Ctx) error {
	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", "/login")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Render("partials/players", fiber.Map{
		"Title": "Players",
	}, "layouts/base")
}

func (h *PlayersHandler) GetFormPlayers(c fiber.Ctx) error {
	return c.Render("partials/form-players", fiber.Map{})
}

func (h *PlayersHandler) GetFormToggle(c fiber.Ctx) error {
	return c.Render("partials/form-toggle-button", nil)
}

func (h *PlayersHandler) UpdatePlayers(c fiber.Ctx) error {
	var bodies []struct {
		ID        uuid.UUID `json:"id"`
		Name      *string   `json:"name,omitempty"`
		Sex       *string   `json:"sex,omitempty"`
		Country   *string   `json:"country,omitempty"`
		City      *string   `json:"city,omitempty"`
		Birthdate *string   `json:"birthdate,omitempty"`
		Elo       *int16    `json:"elo,omitempty"`
	}

	if err := c.Bind().Body(&bodies); err != nil {
		var single struct {
			ID        uuid.UUID `json:"id"`
			Name      *string   `json:"name,omitempty"`
			Sex       *string   `json:"sex,omitempty"`
			Country   *string   `json:"country,omitempty"`
			City      *string   `json:"city,omitempty"`
			Birthdate *string   `json:"birthdate,omitempty"`
			Elo       *int16    `json:"elo,omitempty"`
		}
		if err := c.Bind().Body(&single); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		bodies = append(bodies, single)
	}

	updatedPlayers := []*players.Players{}

	for _, body := range bodies {
		updates := map[string]interface{}{}
		if body.Name != nil {
			updates["name"] = *body.Name
		}
		if body.Sex != nil {
			updates["sex"] = *body.Sex
		}
		if body.Country != nil {
			updates["country"] = *body.Country
		}
		if body.City != nil {
			updates["city"] = *body.City
		}
		if body.Birthdate != nil {
			updates["birthdate"] = *body.Birthdate
		}
		if body.Elo != nil {
			updates["elo"] = *body.Elo
		}
		updates["updated_at"] = time.Now().UTC()

		updatedPlayer, err := h.service.UpdatePlayers(context.Background(), body.ID, updates)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		updatedPlayers = append(updatedPlayers, updatedPlayer)
	}

	return c.Status(fiber.StatusOK).JSON(updatedPlayers)
}

func (h *PlayersHandler) DeletePlayers(c fiber.Ctx) error {
	var body struct {
		ID string `json:"id"`
	}

	if err := c.Bind().Body(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	uid, err := uuid.Parse(body.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id format")
	}

	if err := h.service.DeletePlayers(context.Background(), uid); err != nil {
		if err.Error() == "player not found" {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.SendStatus(fiber.StatusOK)
}
