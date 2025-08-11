package main

import (
	"fmt"
	"log"
	"time"

	"github.com/chasefleming/elem-go"
	"github.com/chasefleming/elem-go/attrs"
	"github.com/chasefleming/elem-go/htmx"
	"github.com/chasefleming/elem-go/styles"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/storage/sqlite3/v2"
	"github.com/joho/godotenv"
	"github.com/monzork/table-tennis-backend/internal/db"
)

var sqliteStorage = sqlite3.New(sqlite3.Config{
	Database: "../sessions.db",
	Table:    "sessions",
})

var bodyStyle = styles.Props{
	styles.BackgroundColor: "#f4f4f4",
	styles.FontFamily:      "Arial, sans-serif",
	styles.Height:          "100vh",
	styles.Display:         "flex",
	styles.FlexDirection:   "column",
	styles.AlignItems:      "center",
	styles.JustifyContent:  "center",
}

var buttonStyle = styles.Props{
	styles.Padding:         "10px 20px",
	styles.BackgroundColor: "#007BFF",
	styles.Color:           "#fff",
	styles.BorderColor:     "#007BFF",
	styles.BorderRadius:    "5px",
	styles.Margin:          "10px",
	styles.Cursor:          "pointer",
}

func main() {
	godotenv.Load()
	db.Connect()

	app := fiber.New()

	app.Post("/submit", func(c fiber.Ctx) error {
		// Handle POST request data
		type Players struct {
			Name      string `json:"name"`
			Sex       string `json:"sex"`
			Birthdate string `json:"birthdate"`
			Country   string `json:"country"`
			City      string `json:"city"`
		}

		//id := uuid.New() //poner en otro lado

		var p Players
		if err := c.Bind().Body(&p); err != nil {
			log.Println(err)
			return c.Status(500).SendString("Failed to parse request")
		}

		return c.JSON(p) // Respond with the parsed data
	})

	app.Use("/static", static.New("./static/html/index.html"))
	//app.Get("/users", handlers.ListUsers)
	// test := static.New("./public")
	// print("%+v\n", &test)
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
		CookieName:        "__Host-csrf_",
		CookieSecure:      true,
		CookieHTTPOnly:    true,
		CookieSameSite:    "lax",
		CookieSessionOnly: true,
		Extractor:         csrf.FromHeader("X-Csrf-Token"),
		//Session:           store,
	}))

	app.Get("/static", func(c fiber.Ctx) error {
		body := elem.Body(
			attrs.Props{
				attrs.Style: bodyStyle.ToInline(),
			},
			elem.H1(nil, elem.Text("Counter App reloading")),
			elem.Div(attrs.Props{attrs.ID: "count"}, elem.Text("0")),
			elem.Button(
				attrs.Props{
					htmx.HXPost:   "/increment",
					htmx.HXTarget: "#count",
					attrs.Style:   buttonStyle.ToInline(),
				},
				elem.Text("+"),
			),

			elem.Button(
				attrs.Props{
					htmx.HXPost:   "/decrement",
					htmx.HXTarget: "#count",
					attrs.Style:   buttonStyle.ToInline(),
				},
				elem.Text("-"),
			),
		)

		head := elem.Head(nil, elem.Script(attrs.Props{attrs.Src: "https://unpkg.com/htmx.org@1.9.6"}))
		pageContent := elem.Html(nil, head, body)
		html := pageContent.Render()
		c.Type("html")
		return c.SendString(html)
	})

	log.Fatal(app.Listen(":3000"))
}
