package routes

import (
	"github.com/gofiber/fiber/v3"
	userHandler "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"
)

func RegisterPublicRoutes(app fiber.Router, h *userHandler.UserHandler) {
	app.Post("/login", h.Login)
}
