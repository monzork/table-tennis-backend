package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/joho/godotenv"
	"github.com/monzork/table-tennis-backend/internal/db"
)

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
	log.Fatal(app.Listen(":3000"))
}
