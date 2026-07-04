package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// Protected checks the session to see if the user is authenticated.
func Protected(store *session.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		handleUnauthorized := func() error {
			if c.Get("HX-Request") == "true" {
				c.Set("HX-Redirect", "/admin/login")
				return c.SendStatus(fiber.StatusUnauthorized)
			}
			return c.Redirect("/admin/login")
		}

		sess, err := store.Get(c)
		if err != nil {
			return handleUnauthorized()
		}

		auth := sess.Get("authenticated")
		if auth == nil || !auth.(bool) {
			return handleUnauthorized()
		}

		// User is authenticated, proceed
		return c.Next()
	}
}
