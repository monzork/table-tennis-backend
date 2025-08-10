package main

import (
	"context"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/joho/godotenv"
	"github.com/monzork/table-tennis-backend/internal/db"
	//"github.com/monzork/table-tennis-backend/internal/handlers"
	"github.com/monzork/table-tennis-backend/internal/models"
	"log"
)

func main() {
	godotenv.Load()
	db.Connect()
	db.DB.NewInsert().Model(&models.User{Name: "Roger"}).Exec(context.Background())

	app := fiber.New()

	app.Use("/static", static.New("./static/html/index.html"))
	//app.Get("/users", handlers.ListUsers)
	// test := static.New("./public")
	// print("%+v\n", &test)
	log.Fatal(app.Listen(":3000"))
}
