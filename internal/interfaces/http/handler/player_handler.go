package handler

import (
	"context"
	"fmt"
	"table-tennis-backend/internal/application/player"

	"github.com/gofiber/fiber/v2"
)

type PlayerHandler struct {
	registerPlayerUC *player.RegisterPlayerUseCase
	updatePlayerUC   *player.UpdatePlayerUseCase
	deletePlayerUC   *player.DeletePlayerUseCase
	importPlayersUC  *player.ImportPlayersUseCase
}

func NewPlayerHandler(
	uc *player.RegisterPlayerUseCase,
	uuc *player.UpdatePlayerUseCase,
	duc *player.DeletePlayerUseCase,
	iuc *player.ImportPlayersUseCase,
) *PlayerHandler {
	return &PlayerHandler{
		registerPlayerUC: uc,
		updatePlayerUC:   uuc,
		deletePlayerUC:   duc,
		importPlayersUC:  iuc,
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

// ImportTemplate returns a downloadable CSV template with the correct headers.
func (h *PlayerHandler) ImportTemplate(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=\"players_template.csv\"")
	return c.SendString("first_name,last_name,birthdate,gender,country\nJohn,Doe,1995-06-15,M,MEX\nJane,Smith,1998-03-22,F,USA\n")
}
func (h *PlayerHandler) Import(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "no file uploaded")
	}

	f, err := file.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to open file")
	}
	defer f.Close()

	result, err := h.importPlayersUC.Execute(c.Context(), file.Filename, f)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	msg := fmt.Sprintf("Imported %d players", result.Imported)
	if result.Skipped > 0 {
		msg += fmt.Sprintf(", %d skipped", result.Skipped)
	}

	// Return an HTMX-friendly JSON summary; the UI will handle the toast + reload
	return c.JSON(fiber.Map{
		"imported": result.Imported,
		"skipped":  result.Skipped,
		"errors":   result.Errors,
		"message":  msg,
	})
}
