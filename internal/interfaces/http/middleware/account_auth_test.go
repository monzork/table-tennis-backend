package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"table-tennis-backend/internal/interfaces/http/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

func TestAccountProtectedMiddleware(t *testing.T) {
	store := session.New()

	app := fiber.New()

	app.Post("/account-login-session", func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		sess.Set("account_authenticated", true)
		sess.Set("account_id", "acc-123")
		if err := sess.Save(); err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendString("logged in")
	})

	app.Post("/admin-login-session", func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		sess.Set("authenticated", true)
		if err := sess.Save(); err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendString("admin logged in")
	})

	app.Post("/account-login-session-missing-id", func(c *fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		sess.Set("account_authenticated", true)
		if err := sess.Save(); err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendString("logged in without id")
	})

	protected := app.Group("/account-area", middleware.AccountProtected(store))
	protected.Get("/dashboard", func(c *fiber.Ctx) error {
		accountID, _ := c.Locals("AccountID").(string)
		return c.SendString("dashboard:" + accountID)
	})

	adminProtected := app.Group("/admin-area", middleware.Protected(store))
	adminProtected.Get("/dashboard", func(c *fiber.Ctx) error {
		return c.SendString("admin dashboard")
	})

	t.Run("unauthenticated request redirects to account login", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/account-area/dashboard", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusFound {
			t.Errorf("expected status %d, got %d", http.StatusFound, resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "/account/login" {
			t.Errorf("expected Location '/account/login', got %q", loc)
		}
	})

	t.Run("unauthenticated HX-Request returns 401 with HX-Redirect", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/account-area/dashboard", nil)
		req.Header.Set("HX-Request", "true")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		}
		if hx := resp.Header.Get("HX-Redirect"); hx != "/account/login" {
			t.Errorf("expected HX-Redirect '/account/login', got %q", hx)
		}
	})

	t.Run("authenticated account session allows request and exposes AccountID", func(t *testing.T) {
		loginResp, err := app.Test(httptest.NewRequest(http.MethodPost, "/account-login-session", nil))
		if err != nil {
			t.Fatalf("login failed: %v", err)
		}
		cookies := loginResp.Header.Values("Set-Cookie")
		if len(cookies) == 0 {
			t.Fatal("expected Set-Cookie header")
		}

		req := httptest.NewRequest(http.MethodGet, "/account-area/dashboard", nil)
		for _, c := range cookies {
			req.Header.Add("Cookie", c)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("account session with no account_id is unauthorized", func(t *testing.T) {
		loginResp, err := app.Test(httptest.NewRequest(http.MethodPost, "/account-login-session-missing-id", nil))
		if err != nil {
			t.Fatalf("login failed: %v", err)
		}
		cookies := loginResp.Header.Values("Set-Cookie")

		req := httptest.NewRequest(http.MethodGet, "/account-area/dashboard", nil)
		for _, c := range cookies {
			req.Header.Add("Cookie", c)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusFound {
			t.Errorf("expected status %d, got %d", http.StatusFound, resp.StatusCode)
		}
	})

	t.Run("an admin-only session never satisfies AccountProtected", func(t *testing.T) {
		loginResp, err := app.Test(httptest.NewRequest(http.MethodPost, "/admin-login-session", nil))
		if err != nil {
			t.Fatalf("admin login failed: %v", err)
		}
		cookies := loginResp.Header.Values("Set-Cookie")

		req := httptest.NewRequest(http.MethodGet, "/account-area/dashboard", nil)
		for _, c := range cookies {
			req.Header.Add("Cookie", c)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusFound {
			t.Errorf("expected admin session to be rejected by AccountProtected, got status %d", resp.StatusCode)
		}
	})

	t.Run("an account-only session never satisfies admin Protected", func(t *testing.T) {
		loginResp, err := app.Test(httptest.NewRequest(http.MethodPost, "/account-login-session", nil))
		if err != nil {
			t.Fatalf("account login failed: %v", err)
		}
		cookies := loginResp.Header.Values("Set-Cookie")

		req := httptest.NewRequest(http.MethodGet, "/admin-area/dashboard", nil)
		for _, c := range cookies {
			req.Header.Add("Cookie", c)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusFound {
			t.Errorf("expected account session to be rejected by admin Protected, got status %d", resp.StatusCode)
		}
	})
}
