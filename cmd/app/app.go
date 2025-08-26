package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/favicon"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v2"
	"github.com/joho/godotenv"
	"github.com/monzork/table-tennis-backend/internal/db"
	"github.com/uptrace/bun"

	playersService "github.com/monzork/table-tennis-backend/internal/domain/players"
	userService "github.com/monzork/table-tennis-backend/internal/domain/user"
	securityHandler "github.com/monzork/table-tennis-backend/internal/infrastructure/security"
	sessionHandler "github.com/monzork/table-tennis-backend/internal/infrastructure/session"
	Repos "github.com/monzork/table-tennis-backend/internal/infrastructure/storage"
	Handlers "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"
	handler "github.com/monzork/table-tennis-backend/internal/transport/http/handlers"
	Routes "github.com/monzork/table-tennis-backend/internal/transport/http/routes"
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

	RegisterRoutes(app, dbConn)

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

	app.Use("/static/css", static.New("./internal/transport/templates/css", static.Config{
		Compress:      true,
		CacheDuration: 86400,
		MaxAge:        86400,
	}))

	// Middlewares
	app.Use(favicon.New(favicon.Config{
		File: "favicon.ico",
		URL:  "/favicon.ico",
	}))
	app.Use(sessionHandler.InitializeSession())
	app.Use(securityHandler.InitializeCSRF())
	app.Use(func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		duration := time.Since(start)
		log.Printf("[%s] %s %s -> %d (%s)",
			time.Now().Format(time.RFC3339),
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			duration,
		)

		return err
	})
	app.Use(SessionMiddleware)

	return app
}

func RegisterRoutes(app *fiber.App, dbConn *bun.DB) {
	api := app.Group("/api")
	buildUserDependencies(app, api, dbConn)
	buildPlayersDependencies(app, api, dbConn)
	buildIndexDependecies(app, api, dbConn)
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	return port
}

func GetCurrentUser(c fiber.Ctx) string {
	username, ok := c.Locals("username").(string)
	if !ok {
		return ""
	}
	return username
}

func SessionMiddleware(c fiber.Ctx) error {
	uri := c.Request().URI()
	if strings.Contains(uri.String(), "/login") {
		return c.Next()
	}
	sess := session.FromContext(c)
	username := sess.Get("username")

	if username == nil {
		if c.Get("HX-Request") == "true" {
			c.Set("HX-Redirect", "/login")
			return c.SendStatus(fiber.StatusOK)
		}

		return c.Redirect().To("/login")
	}

	c.Locals("User", fiber.Map{
		"Username": username,
	})

	return c.Next()
}

func buildUserDependencies(app *fiber.App, api fiber.Router, db *bun.DB) {
	userRepository := Repos.NewSQLiteUserRepository(db)
	userService := userService.NewService(userRepository)
	userHandler := Handlers.NewUserHandler(userService)
	Routes.RegisterUserRoutes(api, app, userHandler)
}

func buildPlayersDependencies(app *fiber.App, api fiber.Router, db *bun.DB) {
	playersRepository := Repos.NewSQLitePlayersRepository(db)
	playersService := playersService.NewService(playersRepository)
	playersHandler := Handlers.NewPlayersHandler(playersService)
	Routes.RegisterPlayersRoutes(app, api, playersHandler)
}

func buildIndexDependecies(app *fiber.App, api fiber.Router, db *bun.DB) {
	// TODO: find a better way to implement this
	indexHandler := handler.NewIndexHandler()
	userRepository := Repos.NewSQLiteUserRepository(db)
	userService := userService.NewService(userRepository)
	userHandler := Handlers.NewUserHandler(userService)
	Routes.RegisterPublicRoutes(app, api, indexHandler, userHandler)
}
