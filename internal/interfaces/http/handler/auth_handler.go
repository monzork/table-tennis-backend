package handler

import (
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type AuthHandler struct {
	store     *session.Store
	adminRepo *bun.AdminRepository
}

func NewAuthHandler(store *session.Store, adminRepo *bun.AdminRepository) *AuthHandler {
	return &AuthHandler{store: store, adminRepo: adminRepo}
}

// ShowLogin renders the login page
func (h *AuthHandler) ShowLogin(c *fiber.Ctx) error {
	// If already logged in, redirect to admin
	sess, err := h.store.Get(c)
	if err == nil {
		if auth := sess.Get("authenticated"); auth != nil && auth.(bool) {
			return c.Redirect("/admin")
		}
	}
	// Note: We use the public layout because we don't want the admin nav menu on login
	return c.Render("admin/login", fiber.Map{}, "layouts/public")
}

// Login validates credentials and establishes session
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var body struct {
		Username string `json:"username" form:"username"`
		Password string `json:"password" form:"password"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Render("admin/login", fiber.Map{"Error": "Invalid form submission"}, "layouts/public")
	}

	admin, err := h.adminRepo.GetByUsername(c.Context(), body.Username)
	if err != nil || admin == nil {
		return c.Render("admin/login", fiber.Map{"Error": "Invalid Username or Password"}, "layouts/public")
	}

	if !admin.CheckPassword(body.Password) {
		return c.Render("admin/login", fiber.Map{"Error": "Invalid Username or Password"}, "layouts/public")
	}

	sess, err := h.store.Get(c)
	if err != nil {
		return c.Render("admin/login", fiber.Map{"Error": "Session error"}, "layouts/public")
	}

	sess.Set("authenticated", true)
	if err := sess.Save(); err != nil {
		return c.Render("admin/login", fiber.Map{"Error": "Failed to save session"}, "layouts/public")
	}

	// Use HTMX redirect header if it's an HTMX request, otherwise normal redirect
	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", "/admin")
		return c.SendStatus(200)
	}
	return c.Redirect("/admin")
}

// Logout clears the session
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	sess, err := h.store.Get(c)
	if err == nil {
		sess.Destroy()
	}
	return c.Redirect("/admin/login")
}
