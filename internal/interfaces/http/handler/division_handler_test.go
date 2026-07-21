package handler_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestDivisionHandler(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	loginReq := httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=admin&password=password"))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResp, _ := app.Test(loginReq)

	var sessionCookie string
	for _, v := range loginResp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(v, "session_id=") {
			sessionCookie = strings.Split(v, ";")[0]
		}
	}

	t.Run("Create Division", func(t *testing.T) {
		data := url.Values{}
		data.Set("name", "Division A")
		data.Set("min_elo", "1000")
		data.Set("max_elo", "1500")

		req := httptest.NewRequest("POST", "/divisions", strings.NewReader(data.Encode()))
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

	t.Run("Update Division", func(t *testing.T) {
		var dm bunRepo.DivisionModel
		if err := db.NewSelect().Model(&dm).Where("name = ?", "Division A").Scan(context.Background()); err != nil {
			t.Fatalf("failed to find seeded division: %v", err)
		}

		data := url.Values{}
		data.Set("id", dm.ID)
		data.Set("name", "Division A Updated")
		data.Set("min_elo", "1000")
		data.Set("max_elo", "1500")

		req := httptest.NewRequest("POST", "/divisions", strings.NewReader(data.Encode()))
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

	t.Run("Delete Division", func(t *testing.T) {
		var dm bunRepo.DivisionModel
		err := db.NewSelect().Model(&dm).Where("name = ?", "Division A Updated").Scan(context.Background())
		if err != nil {
			t.Fatalf("failed to find seeded division: %v", err)
		}

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/divisions/%s", dm.ID), bytes.NewReader([]byte{}))
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Show Edit Form - New", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/divisions/edit", nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Show Edit Form - Existing", func(t *testing.T) {
		// The previous test deletes "Division A", so create a fresh one to edit.
		data := url.Values{}
		data.Set("name", "Division B")
		data.Set("min_elo", "500")
		data.Set("max_elo", "900")
		createReq := httptest.NewRequest("POST", "/divisions", strings.NewReader(data.Encode()))
		createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		createReq.Header.Set("Cookie", sessionCookie)
		if _, err := app.Test(createReq); err != nil {
			t.Fatalf("failed to seed division: %v", err)
		}

		var dm bunRepo.DivisionModel
		err := db.NewSelect().Model(&dm).Where("name = ?", "Division B").Scan(context.Background())
		if err != nil {
			t.Fatalf("failed to find seeded division: %v", err)
		}

		req := httptest.NewRequest("GET", fmt.Sprintf("/divisions/%s/edit", dm.ID), nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Delete Division - Error", func(t *testing.T) {
		appErr, dbErr, _, _ := SetupTestApp()
		loginReq := httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=admin&password=password"))
		loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		loginResp, _ := appErr.Test(loginReq)
		var cookie string
		for _, v := range loginResp.Header.Values("Set-Cookie") {
			if strings.HasPrefix(v, "session_id=") {
				cookie = strings.Split(v, ";")[0]
			}
		}
		dbErr.Close()
		req := httptest.NewRequest("DELETE", "/divisions/invalid-id", nil)
		req.Header.Set("Cookie", cookie)

		resp, _ := appErr.Test(req)
		if resp.StatusCode != 500 {
			t.Errorf("expected 500, got %v", resp.StatusCode)
		}
	})

	t.Run("Show Edit Form - Error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/divisions/invalid-id/edit", nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 404 {
			t.Errorf("expected 404, got %v", resp.StatusCode)
		}
	})
}
