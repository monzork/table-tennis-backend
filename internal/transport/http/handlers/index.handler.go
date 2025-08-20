package handler

import (
	"bytes"
	"html/template"

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

	var dashBuf bytes.Buffer
	err := c.App().Config().Views.Render(&dashBuf, "partials/dashboard", fiber.Map{
		"User": fiber.Map{
			"Username": username,
		},
		"Title": "Dashboard",
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
