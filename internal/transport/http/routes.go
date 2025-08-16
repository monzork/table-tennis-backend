package user

import (
	"github.com/gofiber/fiber/v3"
	user "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"
)

func RegisterRoutes(app fiber.Router, h *user.UserHandler) {
	userGroup := app.Group("/user")
	userGroup.Post("/", h.Register)
	userGroup.Post("Login", h.Login)
}
