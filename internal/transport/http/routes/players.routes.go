package routes

import (
	"github.com/gofiber/fiber/v3"
	playersHandler "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"
)

func RegisterPlayersRoutes(app *fiber.App, api fiber.Router, h *playersHandler.PlayersHandler) {
	playersApiRoutes(api, h)
	playersStaticRoutes(app, h)
}

func playersApiRoutes(api fiber.Router, h *playersHandler.PlayersHandler) {
	apiPlayersGroup := api.Group("/players")
	apiPlayersGroup.Get("/", h.GetAllPlayers)
	apiPlayersGroup.Delete("/:id", h.DeletePlayers)
	apiPlayersGroup.Post("/", h.RegisterPlayers)
}

func playersStaticRoutes(app *fiber.App, h *playersHandler.PlayersHandler) {
	playersGroup := app.Group("/players")
	playersGroup.Put("/", h.UpdatePlayers)
	playersGroup.Get("/", h.ShowPlayersTab)
	playersGroup.Get("/form", h.GetFormPlayers)
	playersGroup.Get("/form/:id", h.GetFormPlayers)
	playersGroup.Get("/form-toggle", h.GetFormToggle)
}
