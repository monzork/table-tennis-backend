package handler_test

import (
	"context"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	divisionDomain "table-tennis-backend/internal/domain/division"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestAdminHandler(t *testing.T) {
	app, db, _, err := SetupTestApp()
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

		// Test admin pages
		adminEndpoints := []string{
			"/admin/players",
			"/admin/tournaments",
			"/admin/divisions",
			"/admin/new-player-field",
			"/admin/close-modal",
			"/admin/tournaments/division-select",
		}

		for _, ep := range adminEndpoints {
			t.Run("Access "+ep, func(t *testing.T) {
				req := httptest.NewRequest("GET", ep, nil)
				req.Header.Set("Cookie", sessionCookie)
				resp, err := app.Test(req)
				if err != nil {
					t.Fatalf("test request failed: %v", err)
				}
				if resp.StatusCode != 200 {
					t.Errorf("expected 200 OK for %s, got %v", ep, resp.StatusCode)
				}
			})
		}

		t.Run("Division select only offers gender-specific divisions", func(t *testing.T) {
			divisionRepo := bunRepo.NewDivisionRepository(db)
			ctx := context.Background()

			legacy, _ := divisionDomain.NewDivision("legacy-both-test", "Legacy Gender-Agnostic Division", 900, 1500, nil, "singles", "#000000")
			if err := divisionRepo.Save(ctx, legacy); err != nil {
				t.Fatalf("failed to seed legacy division: %v", err)
			}

			maxElo := int16(2000)
			gendered, _ := divisionDomain.NewDivision("gendered-male-test", "1st Division (Men) Test", 901, 1600, &maxElo, "singles", "#000000")
			gendered.Gender = "M"
			if err := divisionRepo.Save(ctx, gendered); err != nil {
				t.Fatalf("failed to seed gendered division: %v", err)
			}

			req := httptest.NewRequest("GET", "/admin/tournaments/division-select", nil)
			req.Header.Set("Cookie", sessionCookie)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("test request failed: %v", err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			html := string(body)

			if !strings.Contains(html, "1st Division (Men) Test") {
				t.Errorf("expected gender-specific division to be offered, response was: %s", html)
			}
			if strings.Contains(html, "Legacy Gender-Agnostic Division") {
				t.Errorf("expected gender-agnostic division to be excluded from new-tournament picker, response was: %s", html)
			}
		})
	})
}
