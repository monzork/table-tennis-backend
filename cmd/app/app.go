package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/favicon"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
	"github.com/monzork/table-tennis-backend/internal/db"
	userDomain "github.com/monzork/table-tennis-backend/internal/domain/user"

	securityHandler "github.com/monzork/table-tennis-backend/internal/infrastructure/security"
	sessionHandler "github.com/monzork/table-tennis-backend/internal/infrastructure/session"
	userRepo "github.com/monzork/table-tennis-backend/internal/infrastructure/storage"
	userTransport "github.com/monzork/table-tennis-backend/internal/transport/http"
	userHandler "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"

	"github.com/uptrace/bun"
)

func Run() error {
	loadEnv()
	dbConn := initDB()
	defer dbConn.Close()

	app := initApp()
	app.Use(func(c fiber.Ctx) error {
		token := csrf.TokenFromContext(c)
		c.Locals("CSRF", token)
		return c.Next()
	})
	registerRoutes(app, dbConn)

	port := getPort()
	log.Println("Server running on port", port)
	return app.Listen(":" + port)
}

func loadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}
}

func initDB() *bun.DB {
	return db.Connect()
}

func initApp() *fiber.App {
	engine := html.New("./internal/transport/templates", ".html")

	app := fiber.New(fiber.Config{
		AppName:           "TTN",
		Views:             engine,
		PassLocalsToViews: true,
	})

	// Middlewares
	app.Use(favicon.New(favicon.Config{
		File: "favicon.ico",
		URL:  "/favicon.ico",
	}))
	app.Use(sessionHandler.InitializeSession())
	app.Use(securityHandler.InitializeCSRF())

	return app
}

func registerRoutes(app *fiber.App, dbConn *bun.DB) {
	// Public routes
	app.Get("/login", showLogin)
	app.Get("/", func(c fiber.Ctx) error { return c.Redirect().To("/dashboard") })

	app.Get("/dashboard", showDashboard)
	app.Post("/logout", logout)

	// API routes
	api := app.Group("/api")
	buildUserDependencies(api, dbConn)
}

func buildUserDependencies(api fiber.Router, dbConn *bun.DB) {
	repo := userRepo.NewSQLiteUserRepository(dbConn)
	service := userDomain.NewService(repo)
	handler := userHandler.NewUserHandler(service)
	userTransport.RegisterRoutes(api, handler)
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	return port
}

func showLogin(c fiber.Ctx) error {
	sess := session.FromContext(c)
	fmt.Printf("%v", c.Locals("CSRF"))
	if sess.Get("username") != nil {
		return c.Redirect().To("/dashboard")
	}
	return c.Render("pages/login", fiber.Map{
		"Title": "Login",
	}, "layouts/base")
}

func showDashboard(c fiber.Ctx) error {
	sess := session.FromContext(c)
	username := sess.Get("username")
	userID := sess.Get("user_id")
	if username == nil {
		if c.Get("HX-Request") == "true" {
			c.Set("HX-Redirect", "/login")
			return c.SendStatus(fiber.StatusOK)
		}
		return c.Redirect().To("/login")
	}

	return c.Render("pages/dashboard", fiber.Map{
		"Title": "Dashboard",
		"User": fiber.Map{
			"ID":       userID,
			"Username": username,
		},
	}, "layouts/base")
}

func logout(c fiber.Ctx) error {
	sess := session.FromContext(c)
	sess.Destroy()

	c.Set("HX-Redirect", "/login")
	return c.SendStatus(fiber.StatusOK)
}
