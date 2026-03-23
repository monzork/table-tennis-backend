package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// Protected checks the session to see if the user is authenticated.
func Protected(store *session.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return c.Redirect("/admin/login")
		}

		auth := sess.Get("authenticated")
		if auth == nil || !auth.(bool) {
			// Save intended destination for post-login redirect? We won't overcomplicate it.
			return c.Redirect("/admin/login")
		}

		// User is authenticated, proceed
		return c.Next()
	}
}
