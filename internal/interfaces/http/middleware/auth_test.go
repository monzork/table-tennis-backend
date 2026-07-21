package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"table-tennis-backend/internal/interfaces/http/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

func TestProtectedMiddleware(t *testing.T) {
	store := session.New()

	app := fiber.New()

	// Handler to seed session authentication state
	app.Post("/login-session", func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		sess.Set("authenticated", true)
		if err := sess.Save(); err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendString("logged in")
	})

	app.Post("/login-session-false", func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		sess.Set("authenticated", false)
		if err := sess.Save(); err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendString("logged in false")
	})

	protected := app.Group("/protected", middleware.Protected(store))
	protected.Get("/dashboard", func(c *fiber.Ctx) error {
		return c.SendString("dashboard content")
	})

	t.Run("Unauthenticated request redirects to login", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected/dashboard", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusFound {
			t.Errorf("expected status %d, got %d", http.StatusFound, resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "/admin/login" {
			t.Errorf("expected Location header '/admin/login', got %q", loc)
		}
	})

	t.Run("Unauthenticated HX-Request returns 401 with HX-Redirect header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected/dashboard", nil)
		req.Header.Set("HX-Request", "true")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		}
		if hxRedir := resp.Header.Get("HX-Redirect"); hxRedir != "/admin/login" {
			t.Errorf("expected HX-Redirect header '/admin/login', got %q", hxRedir)
		}
	})

	t.Run("Authenticated session allows request", func(t *testing.T) {
		// First perform login request to acquire session cookie
		reqLogin := httptest.NewRequest(http.MethodPost, "/login-session", nil)
		respLogin, err := app.Test(reqLogin)
		if err != nil {
			t.Fatalf("login failed: %v", err)
		}

		cookies := respLogin.Header.Values("Set-Cookie")
		if len(cookies) == 0 {
			t.Fatal("expected Set-Cookie header from session store")
		}

		// Subsequent protected request with cookie
		reqProtected := httptest.NewRequest(http.MethodGet, "/protected/dashboard", nil)
		for _, cookie := range cookies {
			reqProtected.Header.Add("Cookie", cookie)
		}

		respProtected, err := app.Test(reqProtected)
		if err != nil {
			t.Fatalf("protected request failed: %v", err)
		}

		if respProtected.StatusCode != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, respProtected.StatusCode)
		}
	})

	t.Run("Session authenticated flag set to false is unauthorized", func(t *testing.T) {
		reqLogin := httptest.NewRequest(http.MethodPost, "/login-session-false", nil)
		respLogin, err := app.Test(reqLogin)
		if err != nil {
			t.Fatalf("login failed: %v", err)
		}

		cookies := respLogin.Header.Values("Set-Cookie")

		reqProtected := httptest.NewRequest(http.MethodGet, "/protected/dashboard", nil)
		for _, cookie := range cookies {
			reqProtected.Header.Add("Cookie", cookie)
		}

		respProtected, err := app.Test(reqProtected)
		if err != nil {
			t.Fatalf("protected request failed: %v", err)
		}

		if respProtected.StatusCode != http.StatusFound {
			t.Errorf("expected status %d, got %d", http.StatusFound, respProtected.StatusCode)
		}
	})
}
