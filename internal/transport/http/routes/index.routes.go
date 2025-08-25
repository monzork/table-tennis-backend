package routes

import (
	"github.com/gofiber/fiber/v3"
	handler "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"
)

func RegisterPublicRoutes(app fiber.Router, api fiber.Router, h *handler.IndexHandler, u *handler.UserHandler) {
	app.Get("/", func(c fiber.Ctx) error { return c.Redirect().To("/dashboard") })
	app.Get("/login", h.ShowLogin)
	app.Get("/dashboard", h.ShowDashboard)

	app.Post("/logout", u.Logout)
	api.Post("/login", u.Login)
}
