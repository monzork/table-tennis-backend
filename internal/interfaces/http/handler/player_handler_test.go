package handler_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"bytes"

	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestPlayerHandler(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	// Helper to login and get session cookie
	loginReq := httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=admin&password=password"))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResp, _ := app.Test(loginReq)
	
	var sessionCookie string
	for _, v := range loginResp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(v, "session_id=") {
			sessionCookie = strings.Split(v, ";")[0]
		}
	}

	t.Run("Create Player", func(t *testing.T) {
		data := url.Values{}
		data.Set("firstName", "John")
		data.Set("lastName", "Doe")
		data.Set("birthdate", "1990-01-01")
		data.Set("gender", "M")
		data.Set("country", "USA")

		req := httptest.NewRequest("POST", "/players", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Update Player", func(t *testing.T) {
		// Fetch the player created in the previous step
		var pm bunRepo.PlayerModel
		err := db.NewSelect().Model(&pm).Where("first_name = ?", "John").Scan(context.Background())
		if err != nil {
			t.Fatalf("failed to find seeded player: %v", err)
		}

		data := url.Values{}
		data.Set("firstName", "John Updated")
		data.Set("lastName", "Doe")
		data.Set("birthdate", "1990-01-01")
		data.Set("gender", "M")
		data.Set("country", "USA")

		req := httptest.NewRequest("PUT", fmt.Sprintf("/players/%s", pm.ID.String()), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Delete Player", func(t *testing.T) {
		var pm bunRepo.PlayerModel
		err := db.NewSelect().Model(&pm).Where("first_name = ?", "John Updated").Scan(context.Background())
		if err != nil {
			t.Fatalf("failed to find seeded player: %v", err)
		}

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/players/%s", pm.ID.String()), bytes.NewReader([]byte{}))
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
