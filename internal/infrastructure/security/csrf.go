package infrastructure

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
)

func InitializeCSRF() fiber.Handler {
	return csrf.New(csrf.Config{
		CookieName:        "__Host-csrf__",
		CookieSecure:      true,
		CookieHTTPOnly:    true,
		CookieSameSite:    "lax",
		CookieSessionOnly: true,
		Extractor: csrf.Chain(
			csrf.FromHeader("X-Csrf-Token"),
			csrf.FromForm("csrf_token"),
		),
	})
}
