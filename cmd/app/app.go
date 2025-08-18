package main

import (
	"bytes"
	"html/template"
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/favicon"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
	"github.com/monzork/table-tennis-backend/internal/db"

	securityHandler "github.com/monzork/table-tennis-backend/internal/infrastructure/security"
	sessionHandler "github.com/monzork/table-tennis-backend/internal/infrastructure/session"

	Repos "github.com/monzork/table-tennis-backend/internal/infrastructure/storage"

	userService "github.com/monzork/table-tennis-backend/internal/domain/user"

	playersService "github.com/monzork/table-tennis-backend/internal/domain/players"

	Handlers "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"

	Routes "github.com/monzork/table-tennis-backend/internal/transport/http/routes"

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
	buildPlayersDependencies(api, dbConn)
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
	if sess.Get("username") != nil {
		return c.Redirect().To("/dashboard")
	}

	var buf bytes.Buffer

	err := c.App().Config().Views.Render(&buf, "partials/login", fiber.Map{
		"Title": "Login",
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to render login: " + err.Error())
	}

	loginHTML := template.HTML(buf.String())

	return c.Render("layouts/base", fiber.Map{
		"Title":       "Login",
		"MainContent": loginHTML,
	})
}

func showDashboard(c fiber.Ctx) error {
	sess := session.FromContext(c)
	username := sess.Get("username")

	if username == nil {
		if c.Get("HX-Request") == "true" {
			c.Set("HX-Redirect", "/login")
			return c.SendStatus(fiber.StatusOK)
		}
		return c.Redirect().To("/login")
	}

	var formBuf bytes.Buffer
	err := c.App().Config().Views.Render(&formBuf, "partials/form-players", fiber.Map{}, "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			SendString("Failed to render form: " + err.Error())
	}
	formHTML := template.HTML(formBuf.String())

	var dashBuf bytes.Buffer
	err = c.App().Config().Views.Render(&dashBuf, "partials/dashboard", fiber.Map{
		"User": fiber.Map{
			"Username": username,
		},
		"FormPlayers": formHTML,
		"Title":       "Dashboard",
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			SendString("Failed to render dashboard: " + err.Error())
	}

	return c.Render("layouts/base", fiber.Map{
		"Title":       "Dashboard",
		"User":        fiber.Map{"Username": username}, // navbar info
		"MainContent": template.HTML(dashBuf.String()),
	})
}

func logout(c fiber.Ctx) error {
	sess := session.FromContext(c)
	sess.Destroy()

	c.Set("HX-Redirect", "/login")
	return c.SendStatus(fiber.StatusOK)
}

func buildUserDependencies(api fiber.Router, db *bun.DB) {
	userRepository := Repos.NewSQLiteUserRepository(db)
	userService := userService.NewService(userRepository)
	userHandler := Handlers.NewUserHandler(userService)
	Routes.RegisterPublicRoutes(api, userHandler)
	Routes.RegisterUserRoutes(api, userHandler)
}

func buildPlayersDependencies(api fiber.Router, db *bun.DB) {
	playersRepository := Repos.NewSQLitePlayersRepository(db)
	playersService := playersService.NewService(playersRepository)
	playersHandler := Handlers.NewPlayersHandler(playersService)
	Routes.RegisterPlayersRoutes(api, playersHandler)
}
