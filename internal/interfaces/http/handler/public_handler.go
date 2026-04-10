package handler

import (
	"context"
	"table-tennis-backend/internal/application/player"

	"github.com/gofiber/fiber/v2"
)

type PublicHandler struct {
	registerPlayerUC *player.RegisterPlayerUseCase
}

func NewPublicHandler(uc *player.RegisterPlayerUseCase) *PublicHandler {
	return &PublicHandler{registerPlayerUC: uc}
}

func (h *PublicHandler) ShowSignup(c *fiber.Ctx) error {
	return c.Render("register", fiber.Map{
		"Title": "Join the League",
	})
}

func (h *PublicHandler) Register(c *fiber.Ctx) error {
	var body struct {
		FirstName      string `form:"firstName"`
		LastName       string `form:"lastName"`
		Birthdate      string `form:"birthdate"`
		Country        string `form:"country"`
		Gender         string `form:"gender"`
		WhatsAppNumber string `form:"whatsAppNumber"`
		Honeypot       string `form:"website"` // Honeypot field
	}

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Honeypot check: if 'website' is filled, it's likely a bot
	if body.Honeypot != "" {
		// Return 200 OK to the bot but do nothing
		return c.Render("register-success", fiber.Map{
			"Message": "Thank you for your interest!",
		})
	}

	_, err := h.registerPlayerUC.Execute(
		context.Background(),
		body.FirstName,
		body.LastName,
		body.Birthdate,
		body.Gender,
		body.Country,
		body.WhatsAppNumber,
		500, // Default starting elo
		500,
	)

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("register-success", fiber.Map{
		"Title": "Success",
		"Message": body.FirstName + ", you are registered! Go to the rankings to see your profile.",
	})
}
