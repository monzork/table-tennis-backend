package handler_test

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAdminHandler(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	t.Run("Unauthenticated users cannot access dashboard", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin/", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 302 {
			t.Errorf("expected 302 Redirect to login, got %v", resp.StatusCode)
		}
		if resp.Header.Get("Location") != "/admin/login" {
			t.Errorf("expected redirect to login, got %v", resp.Header.Get("Location"))
		}
	})

	t.Run("Authenticated users can access dashboard", func(t *testing.T) {
		// 1. First login to get a cookie
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

		if sessionCookie == "" {
			t.Fatalf("did not receive session cookie upon login")
		}

		// 2. Access dashboard
		req := httptest.NewRequest("GET", "/admin/", nil)
		req.Header.Set("Cookie", sessionCookie)
		
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})
}
