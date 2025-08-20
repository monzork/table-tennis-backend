package routes

import (
	"github.com/gofiber/fiber/v3"
	UserHandler "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"
)

func RegisterUserRoutes(app fiber.Router, h *UserHandler.UserHandler) {
	userGroup := app.Group("/user")
	userGroup.Post("/", h.Register)
	app.Post("/logout", h.Logout)
}
