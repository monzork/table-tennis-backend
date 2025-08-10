package handlers

import (
	"context"

	"github.com/CloudyKit/jet/v6"
	"github.com/gofiber/fiber/v3"
	"github.com/monzork/table-tennis-backend/internal/db"
	"github.com/monzork/table-tennis-backend/internal/models"
)

var views = jet.NewSet(
	jet.NewOSFileSystemLoader("./internal/templates"),
	jet.InDevelopmentMode(),
)

func ListUsers(c fiber.Ctx) error {
	var users []models.User

	err := db.DB.NewSelect().Model(&users).Scan(context.Background())

	if err != nil {
		return c.Status(500).SendString("Error loading users")
	}

	view, err := views.GetTemplate("users.jet")

	if err != nil {
		return c.Status(500).SendString("template error")
	}

	vars := make(jet.VarMap)
	vars.Set("users", users)

	c.Type("html")
	return view.Execute(c.Response().BodyWriter(), vars, nil)
}
