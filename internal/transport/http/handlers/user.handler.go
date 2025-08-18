package handler

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/monzork/table-tennis-backend/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	service *user.Service
}

func NewUserHandler(service *user.Service) *UserHandler {
	return &UserHandler{
		service: service,
	}
}

func (h *UserHandler) Register(c fiber.Ctx) error {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	hashedPassword, err := HashPassword(body.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	u, err := h.service.RegisterUser(context.Background(), body.Username, hashedPassword)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(u)
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func (h *UserHandler) Login(c fiber.Ctx) error {
	var body struct {
		Username string `json:"username" form:"username"`
		Password string `json:"password" form:"password"`
	}

	if err := c.Bind().Body(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid")
	}

	user, err := h.service.Login(c, body.Username, body.Password)

	if err != nil || user == nil {
		return c.Status(fiber.StatusUnauthorized).Render("partials/login_form", fiber.Map{
			"Error": "Invalidad username or password",
		})
	}

	sess := session.FromContext(c)

	if err := sess.Session.Regenerate(); err != nil {
		return err
	}

	sess.Set("user_id", user.ID.String())
	sess.Set("username", user.Username)

	c.Set("HX-Redirect", "/dashboard")
	return c.SendStatus(fiber.StatusOK)
}
