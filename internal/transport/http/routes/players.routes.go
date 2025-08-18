package routes

import (
	"github.com/gofiber/fiber/v3"
	playersHandler "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"
)

func RegisterPlayersRoutes(app fiber.Router, h *playersHandler.PlayersHandler) {
	playersGroup := app.Group("/players")
	playersGroup.Post("/", h.Register)
}
