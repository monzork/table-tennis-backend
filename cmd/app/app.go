package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/storage/sqlite3/v2"
	"github.com/joho/godotenv"
	"github.com/monzork/table-tennis-backend/internal/db"
	userService "github.com/monzork/table-tennis-backend/internal/domain/user"
	userRepo "github.com/monzork/table-tennis-backend/internal/infrastructure/storage"
	user "github.com/monzork/table-tennis-backend/internal/transport/http"
	userHanlder "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"

	"github.com/uptrace/bun"
)

var sqliteStorage = sqlite3.New(sqlite3.Config{
	Database: "./sessions.db",
	Table:    "sessions",
})

func Run() error {
	print("test")

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Connect to DB
	db := db.Connect()
	defer db.Close()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "TTN",
	})

	api := app.Group("/api")

	buildUserDependencies(api, db)

	// Global middleware
	store := initializeSession()
	csrf := initializeCSRF()

	app.Use(csrf)
	app.Use(store)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Println("Server running on port", port)

	return app.Listen(":" + port)
}

func buildUserDependencies(api fiber.Router, db *bun.DB) {
	userRepository := userRepo.NewSQLiteUserRepository(db)
	userService := userService.NewService(userRepository)
	userHandler := userHanlder.NewUserHandler(userService)
	user.RegisterRoutes(api, userHandler)
}

func initializeSession() fiber.Handler {
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

func initializeCSRF() fiber.Handler {
	return csrf.New(csrf.Config{
		CookieName:        "csrf_Protection",
		CookieSecure:      true,
		CookieHTTPOnly:    true,
		CookieSameSite:    "lax",
		CookieSessionOnly: true,
		Extractor: csrf.Chain(
			csrf.FromHeader("X-Csrf-Token"),
			csrf.FromForm("csrf_token")),
	})
}
