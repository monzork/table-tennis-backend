package routes

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	playersHandler "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"
)

func RegisterPlayersRoutes(app *fiber.App, api fiber.Router, h *playersHandler.PlayersHandler) {
	playersGroup := app.Group("/players")
	apiPlayersGroup := api.Group("/players")
	apiPlayersGroup.Get("/", h.GetAllPlayers)
	playersGroup.Post("/", h.Register)
	playersGroup.Get("/", h.ShowPlayersTab)
	playersGroup.Get("/form", h.GetFormPlayers)
	playersGroup.Get("/form-toggle", h.GetFormToggle)

	fmt.Printf("%v", api)
}
