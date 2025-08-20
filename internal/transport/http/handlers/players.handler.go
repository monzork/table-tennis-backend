package handler

import (
	"bytes"
	"context"
	"html/template"

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

	return c.Render("players-row", player)
}

func (h *PlayersHandler) GetAllPlayers(c fiber.Ctx) error {
	playersList, err := h.service.GetAllPlayers(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Render the partial template with players
	var buf bytes.Buffer
	if err := c.App().Config().Views.Render(&buf, "partials/players-list", fiber.Map{
		"Players": playersList,
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	return c.Type("html").SendString(buf.String())
}

func (h *PlayersHandler) ShowPlayersTab(c fiber.Ctx) error {
	var formBuf bytes.Buffer
	_ = c.App().Config().Views.Render(&formBuf, "partials/form-players", fiber.Map{})
	formHTML := template.HTML(formBuf.String())

	// Render Players tab
	var tabBuf bytes.Buffer
	username := 0
	_ = c.App().Config().Views.Render(&tabBuf, "partials/players", fiber.Map{
		"User":        fiber.Map{"Username": username},
		"FormPlayers": formHTML,
	})

	return c.Render("layouts/base", fiber.Map{
		"Title":       "Players",
		"User":        fiber.Map{"Username": username},
		"MainContent": template.HTML(tabBuf.String()),
	})
}

func (h *PlayersHandler) GetFormPlayers(c fiber.Ctx) error {
	return c.Render("partials/form-players", fiber.Map{})
}

func (h *PlayersHandler) GetFormToggle(c fiber.Ctx) error {
	return c.Render("partials/form-toggle-button", nil)
}
