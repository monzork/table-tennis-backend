package handler_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func TestLogoHomeLink_AdminVsPublic(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	t.Run("anonymous visitor: logo links to public dashboard", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/rankings/singles", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, `href="/dashboard"`) {
			t.Errorf("expected logo to link to /dashboard for anonymous visitor, body missing it")
		}
		if strings.Contains(body, `href="/admin"`) {
			t.Errorf("did not expect logo to link to /admin for anonymous visitor")
		}
	})

	t.Run("logged-in admin: logo links to /admin", func(t *testing.T) {
		loginReq := httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=admin&password=password"))
		loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		loginResp, err := app.Test(loginReq)
		if err != nil {
			t.Fatalf("login request failed: %v", err)
		}
		var sessionCookie string
		for _, v := range loginResp.Header.Values("Set-Cookie") {
			if strings.HasPrefix(v, "session_id=") {
				sessionCookie = strings.Split(v, ";")[0]
			}
		}

		req := httptest.NewRequest("GET", "/rankings/singles", nil)
		req.Header.Set("Cookie", sessionCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, `href="/admin"`) {
			t.Errorf("expected logo to link to /admin for logged-in admin, body missing it")
		}
	})
}
