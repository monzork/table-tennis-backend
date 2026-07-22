package handler_test

import (
	"bytes"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAuthHandler(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	t.Run("GET /admin/login", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin/login", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %v", resp.StatusCode)
		}
	})

	t.Run("POST /admin/login with valid credentials", func(t *testing.T) {
		data := url.Values{}
		data.Set("username", "admin")
		data.Set("password", "password")

		req := httptest.NewRequest("POST", "/admin/login", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 302 {
			t.Errorf("expected 302 Found, got %v", resp.StatusCode)
		}

		location := resp.Header.Get("Location")
		if location != "/admin" {
			t.Errorf("expected redirect to /admin, got %v", location)
		}
	})

	t.Run("POST /admin/login with invalid credentials", func(t *testing.T) {
		data := url.Values{}
		data.Set("username", "admin")
		data.Set("password", "wrongpass")

		req := httptest.NewRequest("POST", "/admin/login", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		// It returns 200 OK rendering the login page with Error block
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %v", resp.StatusCode)
		}
	})

	t.Run("POST /admin/logout", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/admin/logout", bytes.NewReader([]byte{}))
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 302 {
			t.Errorf("expected 302 Redirect, got %v", resp.StatusCode)
		}

		location := resp.Header.Get("Location")
		if location != "/admin/login" {
			t.Errorf("expected redirect to /admin/login, got %v", location)
		}
	})

	t.Run("POST /admin/login with HTMX request header", func(t *testing.T) {
		data := url.Values{}
		data.Set("username", "admin")
		data.Set("password", "password")

		req := httptest.NewRequest("POST", "/admin/login", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK for HX-Request, got %v", resp.StatusCode)
		}
		if resp.Header.Get("HX-Redirect") != "/admin" {
			t.Errorf("expected HX-Redirect to /admin, got %v", resp.Header.Get("HX-Redirect"))
		}
	})

	t.Run("GET /admin/login when already authenticated", func(t *testing.T) {
		// First login
		data := url.Values{}
		data.Set("username", "admin")
		data.Set("password", "password")

		loginReq := httptest.NewRequest("POST", "/admin/login", strings.NewReader(data.Encode()))
		loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		loginResp, err := app.Test(loginReq)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		var sessionCookie string
		for _, v := range loginResp.Header.Values("Set-Cookie") {
			if strings.HasPrefix(v, "session_id=") {
				sessionCookie = strings.Split(v, ";")[0]
			}
		}

		// Then try to access login page
		req := httptest.NewRequest("GET", "/admin/login", nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 302 {
			t.Errorf("expected 302 Redirect to /admin, got %v", resp.StatusCode)
		}
	})

	t.Run("POST /admin/login with invalid form data (body parser error)", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/admin/login", strings.NewReader("invalid body"))
		req.Header.Set("Content-Type", "application/json") // sending plain text but claiming it's json

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK rendering error page, got %v", resp.StatusCode)
		}
	})

	t.Run("POST /admin/login with non-existent user", func(t *testing.T) {
		data := url.Values{}
		data.Set("username", "does_not_exist")
		data.Set("password", "password")

		req := httptest.NewRequest("POST", "/admin/login", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK rendering error page, got %v", resp.StatusCode)
		}
	})
}
