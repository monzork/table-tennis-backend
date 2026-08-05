package handler_test

import (
	"net/http/httptest"
	"testing"
)

func TestDashboardHandler_Public(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	req := httptest.NewRequest("GET", "/dashboard", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 OK, got %v", resp.StatusCode)
	}
}
