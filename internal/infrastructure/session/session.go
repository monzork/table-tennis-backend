package infraestructure

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/storage/sqlite3/v2"
)

var sqliteStorage = sqlite3.New(sqlite3.Config{
	Database: "./sessions.db",
	Table:    "sessions",
})

func InitializeSession() fiber.Handler {
	return session.New(session.Config{
		Storage:         sqliteStorage,
		CookieSecure:    true,
		CookieHTTPOnly:  true,
		CookieSameSite:  "lax",
		IdleTimeout:     24 * time.Hour,
		AbsoluteTimeout: 24 * time.Hour,
		ErrorHandler: func(c fiber.Ctx, err error) {
			fmt.Printf("Session error: %v", err)
		},
	})
}
