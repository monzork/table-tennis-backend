package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/infrastructure/identity"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	"table-tennis-backend/internal/interfaces/http/i18n"
	"table-tennis-backend/internal/interfaces/http/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/utils"
	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	idgen.Register(identity.NewUUIDGenerator())

	// Load .env file (ignored in production where env vars are set directly)
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found or failed to load")
	}

	cfg := LoadConfig()

	bun.Connect()

	store := session.New(session.Config{
		Expiration:     24 * time.Hour,
		KeyLookup:      "cookie:session_id",
		CookieSecure:   os.Getenv("DATABASE_URL") != "", // secure in production (PostgreSQL = prod)
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
	})

	authMiddleware := middleware.Protected(store)
	c := NewContainer(store, cfg)

	engine := SetupTemplateEngine()
	app := fiber.New(fiber.Config{
		Views:             engine,
		PassLocalsToViews: true,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		// Custom error handler — returns JSON for API calls, HTML for browser requests
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			msg := "Internal Server Error"
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				msg = e.Message
			}
			slog.Error("HTTP error", "status", code, "path", ctx.Path(), "msg", msg, "ip", ctx.IP())

			if code == fiber.StatusNotFound {
				if ctx.Get("HX-Request") != "" {
					return ctx.Status(code).SendString(msg)
				}
				return ctx.Status(code).Render("errors/404", fiber.Map{"Message": msg})
			}
			if ctx.Get("HX-Request") != "" {
				return ctx.Status(code).SendString(msg)
			}
			return ctx.Status(code).Render("errors/500", fiber.Map{"Message": msg})
		},
	})

	// Enable Security Headers
	app.Use(helmet.New())

	// Enable CORS
	allowedOrigins := os.Getenv("CORS_ORIGIN")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000, http://127.0.0.1:3000, https://table-tennis-backend-8isa.onrender.com"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, X-Csrf-Token",
		AllowCredentials: true,
	}))

	// Enable CSRF
	app.Use(csrf.New(csrf.Config{
		KeyLookup:      "header:X-Csrf-Token",
		CookieName:     "csrf_",
		CookieSameSite: "Lax",
		CookieSecure:   os.Getenv("DATABASE_URL") != "",
		CookieHTTPOnly: true,
		Expiration:     1 * time.Hour,
		KeyGenerator:   utils.UUID,
		Extractor:      csrf.CsrfFromHeader("X-Csrf-Token"),
		Session:        store,
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			// If CSRF validation fails (e.g., token expired or server restarted),
			// clear the invalid cookie and force the client to reload the page
			// to generate a fresh token instead of locking them out.
			ctx.ClearCookie("csrf_")
			if ctx.Get("HX-Request") == "true" {
				ctx.Set("HX-Redirect", ctx.OriginalURL())
				return ctx.SendStatus(200)
			}
			return ctx.Redirect(ctx.OriginalURL())
		},
	}))

	// Pass CSRF token to all templates
	app.Use(func(c *fiber.Ctx) error {
		// First try cookie
		token := c.Cookies("csrf_")
		
		// If empty, try context key
		if token == "" {
			if t, ok := c.Locals(csrf.ConfigDefault.ContextKey).(string); ok {
				token = t
			}
		}

		c.Locals("CSRFToken", token)
		return c.Next()
	})

	// Enable Gzip/Brotli compression for all text-based responses (HTML, CSS, JS, JSON)
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Static files with caching headers
	app.Static("/static", "./static", fiber.Static{
		MaxAge:        86400, // 24 hours
		CacheDuration: 10 * time.Minute,
	})
	app.Static("/open_tdm.jpeg", "./open_tdm.jpeg", fiber.Static{
		MaxAge: 86400,
	})
	app.Static("/favicon.ico", "./favicon.ico", fiber.Static{
		MaxAge: 86400,
	})

	// Global Translation Middleware
	app.Use(func(ctx *fiber.Ctx) error {
		lang := ctx.Cookies("lang")
		if lang != "es" && lang != "en" {
			lang = "en"
		}
		ctx.Locals("T", i18n.PrecomputedMaps[lang])
		ctx.Locals("Lang", lang)
		return ctx.Next()
	})

	SetupRoutes(app, c, authMiddleware)

	go func() {
		addr := ":" + cfg.Port
		slog.Info("Starting server", "addr", addr)
		if err := app.Listen(addr); err != nil {
			slog.Error("Server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")
	if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
		slog.Error("Server forced to shutdown", "err", err)
	}
	slog.Info("Server exited")
}
