package routes

import (
	"github.com/gofiber/fiber/v3"
	handler "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"
)

func RegisterPublicRoutes(app fiber.Router, h *handler.IndexHandler, u *handler.UserHandler) {
	app.Get("/", func(c fiber.Ctx) error { return c.Redirect().To("/dashboard") })
	app.Post("/login", u.Login)
	app.Get("/login", h.ShowLogin)
	app.Get("/dashboard", h.ShowDashboard)
}
