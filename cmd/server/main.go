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
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	idgen.Register(identity.NewUUIDGenerator())
	// Load .env file
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found or failed to load")
	}

	bun.Connect()

	store := session.New()
	authMiddleware := middleware.Protected(store)
	c := NewContainer(store)

	engine := SetupTemplateEngine()
	app := fiber.New(fiber.Config{
		Views:             engine,
		PassLocalsToViews: true,
	})

	app.Static("/static", "./static")
	app.Static("/open_tdm.jpeg", "./open_tdm.jpeg")
	
	// Global Translation Middleware
	app.Use(func(ctx *fiber.Ctx) error {
		lang := ctx.Cookies("lang")
		if lang != "es" && lang != "en" {
			lang = "en"
		}
		m := make(map[string]string)
		for k := range i18n.Translations["en"] {
			m[k] = i18n.T(lang, k)
		}
		ctx.Locals("T", m)
		ctx.Locals("Lang", lang)
		return ctx.Next()
	})

	SetupRoutes(app, c, authMiddleware)

	go func() {
		if err := app.Listen(":8080"); err != nil {
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
