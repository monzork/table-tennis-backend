package main

import (
	"fmt"
	"log"
	"time"

	// "github.com/chasefleming/elem-go"
	// "github.com/chasefleming/elem-go/attrs"
	// "github.com/chasefleming/elem-go/htmx"
	// "github.com/chasefleming/elem-go/styles"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/storage/sqlite3/v2"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/monzork/table-tennis-backend/internal/db"
)

var sqliteStorage = sqlite3.New(sqlite3.Config{
	Database: "./sessions.db",
	Table:    "sessions",
})

func main() {
	godotenv.Load()
	db.Connect()

	app := fiber.New()

	app.Post("/submit", func(c fiber.Ctx) error {
		// Handle POST request data
		type Players struct {
			ID        uuid.UUID
			Name      string `json:"name"`
			Sex       string `json:"sex"`
			Birthdate string `json:"birthdate"`
			Country   string `json:"country"`
			City      string `json:"city"`
		}

		var p Players
		p.ID = uuid.New() //poner en otro lado
		if err := c.Bind().Body(&p); err != nil {
			log.Println(err)
			fmt.Printf("%v", err)
			return c.Status(500).SendString("Failed to parse request")
		}

		return c.JSON(p) // Respond with the parsed data
	})
	store := session.New(session.Config{
		Storage:         sqliteStorage,
		CookieSecure:    true,
		CookieHTTPOnly:  true,
		CookieSameSite:  "lax",
		IdleTimeout:     24 * time.Hour,
		AbsoluteTimeout: 24 * time.Hour,
		//KeyLookup:       "cookie:session_id",
		ErrorHandler: func(c fiber.Ctx, err error) {
			fmt.Printf("Session error: %v", err)
		},
	})

	app.Use(store)
	app.Use(csrf.New(csrf.Config{
		CookieName:        "csrf_Protection",
		CookieSecure:      true,
		CookieHTTPOnly:    true,
		CookieSameSite:    "lax",
		CookieSessionOnly: true,
		Extractor: csrf.Chain(
			csrf.FromHeader("X-Csrf-Token"),
			csrf.FromForm("csrf_token")),
	}))

	app.Route("/login")

	app.Post("/login", func(c fiber.Ctx) error {
		sess := session.FromContext(c)

		fmt.Printf("%v", sess)

		username := c.FormValue("username")
		password := c.FormValue("password")

		if username == "" || password == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Missing username or password")
		}

		sess.Set("username", username)
		sess.Set("authenticated", true)

		if err := sess.Session.Save(); err != nil {
			return err
		}

		c.Set("HX-Redirect", "/dashboard")
		return c.SendStatus(fiber.StatusOK)
	})

	// app.Get("/login", func(c fiber.Ctx) error {
	// 	token := csrf.TokenFromContext(c)
	// 	c.Type("html")
	// 	return c.SendString(fmt.Sprintf(`
	// 	<!DOCTYPE html>
	// 	<html lang="en">
	// 	<head>
	// 		<meta charset="UTF-8" />
	// 		<title>Login</title>
	// 		<script src="https://unpkg.com/htmx.org@1.9.3"></script>
	// 	</head>
	// 	<body>
	// 		<h2>Login Form</h2>
	// 		<form hx-post="/login" hx-target="#response" hx-swap="innerHTML">
	// 			<input type="hidden" name="csrf_token" value="%s" />
	// 			<input name="username" placeholder="Username" required />
	// 			<input name="password" type="password" placeholder="Password" required />
	// 			<button type="submit">Login</button>
	// 		</form>
	// 		<div id="response"></div>
	// 	</body>
	// 	</html>
	// 	`, token))
	// })

	log.Fatal(app.Listen(":3000"))
}
