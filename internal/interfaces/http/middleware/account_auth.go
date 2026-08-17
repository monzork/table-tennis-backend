package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// AccountProtected checks the session for a guardian-account login,
// structurally independent of the admin session key. This is what
// guarantees "accounts can't reach /admin/*" (and vice versa) rather than a
// route-level blocklist: an admin session never satisfies this guard, and an
// account session never satisfies middleware.Protected.
func AccountProtected(store *session.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		handleUnauthorized := func() error {
			if c.Get("HX-Request") == "true" {
				c.Set("HX-Redirect", "/account/login")
				return c.SendStatus(fiber.StatusUnauthorized)
			}
			return c.Redirect("/account/login")
		}

		sess, err := store.Get(c)
		if err != nil {
			return handleUnauthorized()
		}

		auth := sess.Get("account_authenticated")
		if auth == nil || !auth.(bool) {
			return handleUnauthorized()
		}

		accountID := sess.Get("account_id")
		if accountID == nil || accountID.(string) == "" {
			return handleUnauthorized()
		}
		c.Locals("AccountID", accountID.(string))

		return c.Next()
	}
}
