package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

type IndexHandler struct{}

func NewIndexHandler() *IndexHandler {
	return &IndexHandler{}
}

func (h *IndexHandler) ShowLogin(c fiber.Ctx) error {
	sess := session.FromContext(c)
	if sess.Get("username") != nil {
		return c.Redirect().To("/dashboard")
	}

	return c.Render("pages/login", fiber.Map{"Title": "Login"}, "layouts/base")
}

func (h *IndexHandler) ShowDashboard(c fiber.Ctx) error {
	sess := session.FromContext(c)
	username := sess.Get("username")

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
			"Username": username,
		},
	}, "layouts/base")
}
