package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
	"github.com/monzork/table-tennis-backend/internal/db"
	userService "github.com/monzork/table-tennis-backend/internal/domain/user"
	security "github.com/monzork/table-tennis-backend/internal/infrastructure/security"
	session "github.com/monzork/table-tennis-backend/internal/infrastructure/session"
	userRepo "github.com/monzork/table-tennis-backend/internal/infrastructure/storage"
	user "github.com/monzork/table-tennis-backend/internal/transport/http"
	userHanlder "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"

	"github.com/uptrace/bun"
)

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

	app.Use(session.InitializeSession())
	app.Use(security.InitializeCSRF())

	api := app.Group("/api")

	buildUserDependencies(api, db)

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
