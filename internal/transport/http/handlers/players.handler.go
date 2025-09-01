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
		FirstName          string `json:"firstName"`
		LastName           string `json:"lastName"`
		IdentificationType string `json:"identificationType"`
		IdentificationId   string `json:"identificationId"`
		Sex                string `json:"sex"`
		Country            string `json:"country"`
		City               string `json:"city"`
		Birthdate          string `json:"birthdate"`
		Elo                *int16 `json:"elo,omitempty"`
	}

	if err := c.Bind().Body(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	player, err := h.service.RegisterPlayers(
		context.Background(),
		body.FirstName,
		body.LastName,
		body.IdentificationType,
		body.IdentificationId,
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
	playerId := c.Params("id")
	isEdit := playerId != ""
	var player *players.Players
	data := fiber.Map{
		"IsEdit":             isEdit,
		"ID":                 "",
		"FirstName":          "",
		"LastName":           "",
		"IdentificationType": "",
		"IdentificationId":   "",
		"Sex":                "",
		"Country":            "",
		"City":               "",
		"Birthdate":          "",
		"Elo":                1000,
	}

	if isEdit {
		uid, err := uuid.Parse(playerId)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		player, err = h.service.GetByID(c, uid)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "Jugador no encontrado")
		}

		data = fiber.Map{
			"IsEdit":             true,
			"ID":                 player.ID,
			"FirstName":          player.FirstName,
			"LastName":           player.LastName,
			"IdentificationType": player.IdentificationType,
			"IdentificationId":   player.IdentificationId,
			"Sex":                player.Sex,
			"Country":            player.Country,
			"City":               player.City,
			"Birthdate":          player.Birthdate,
			"Elo":                player.Elo,
		}
	}

	return c.Render("partials/form-players", data)
}

func (h *PlayersHandler) GetFormToggle(c fiber.Ctx) error {
	return c.Render("form-toggle-button", nil)
}

func (h *PlayersHandler) UpdatePlayers(c fiber.Ctx) error {
	playerId := c.Params("id")

	if playerId == "" {
		return fiber.ErrBadRequest
	}

	playerUUID, err := uuid.Parse(playerId)
	if err != nil {
		return fiber.ErrBadRequest
	}

	var bodies []struct {
		FirstName          *string `json:"firstName,omitempty"`
		LastName           *string `json:"lastName,omitempty"`
		IdentificationType *string `json:"identificationType,omitempty"`
		IdentificationId   *string `json:"identificationId,omitempty"`
		Sex                *string `json:"sex,omitempty"`
		Country            *string `json:"country,omitempty"`
		City               *string `json:"city,omitempty"`
		Birthdate          *string `json:"birthdate,omitempty"`
		Elo                *int16  `json:"elo,omitempty"`
	}

	if err := c.Bind().Body(&bodies); err != nil {
		var single struct {
			FirstName          *string `json:"firstName,omitempty"`
			LastName           *string `json:"lastName,omitempty"`
			IdentificationType *string `json:"identificationType,omitempty"`
			IdentificationId   *string `json:"identificationId,omitempty"`
			Sex                *string `json:"sex,omitempty"`
			Country            *string `json:"country,omitempty"`
			City               *string `json:"city,omitempty"`
			Birthdate          *string `json:"birthdate,omitempty"`
			Elo                *int16  `json:"elo,omitempty"`
		}
		if err := c.Bind().Body(&single); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		bodies = append(bodies, single)
	}

	updatedPlayers := []*players.Players{}

	for _, body := range bodies {
		updates := map[string]any{}
		if body.FirstName != nil {
			updates["firstName"] = *body.FirstName
		}
		if body.LastName != nil {
			updates["lastName"] = *body.LastName
		}
		if body.IdentificationType != nil {
			updates["identificationType"] = *body.IdentificationType
		}
		if body.IdentificationId != nil {
			updates["identificationId"] = *body.IdentificationId
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

		updatedPlayer, err := h.service.UpdatePlayers(context.Background(), playerUUID, updates)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		updatedPlayers = append(updatedPlayers, updatedPlayer)
	}

	return c.Render("players-row", updatedPlayers[0])
}

func (h *PlayersHandler) DeletePlayers(c fiber.Ctx) error {
	param := struct {
		ID uuid.UUID `uri:"id"`
	}{}

	if err := c.Bind().URI(&param); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id format")
	}

	if err := h.service.DeletePlayers(context.Background(), param.ID); err != nil {
		if err.Error() == "player not found" {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return nil
}
