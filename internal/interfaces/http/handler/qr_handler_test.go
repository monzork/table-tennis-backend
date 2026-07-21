package handler_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"table-tennis-backend/internal/interfaces/http/handler"

	"github.com/gofiber/fiber/v2"
)

type mockQRGenerator struct {
	lastData string
	lastSize int
	err      error
	resBytes []byte
}

func (m *mockQRGenerator) Generate(data string, size int) ([]byte, error) {
	m.lastData = data
	m.lastSize = size
	if m.err != nil {
		return nil, m.err
	}
	if m.resBytes != nil {
		return m.resBytes, nil
	}
	return []byte("fake-png-bytes"), nil
}

func TestQRHandler_Generate(t *testing.T) {
	gen := &mockQRGenerator{}
	h := handler.NewQRHandler(gen)

	app := fiber.New()
	app.Get("/qr", h.Generate)

	t.Run("Missing data parameter returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/qr", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "data parameter is required" {
			t.Errorf("expected body 'data parameter is required', got %q", string(body))
		}
	})

	t.Run("Valid data parameter with default size 250", func(t *testing.T) {
		gen.resBytes = []byte("png-qr-content")
		req := httptest.NewRequest(http.MethodGet, "/qr?data=https://example.com/match/1", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		if resp.Header.Get("Content-Type") != "image/png" {
			t.Errorf("expected Content-Type 'image/png', got %q", resp.Header.Get("Content-Type"))
		}

		if resp.Header.Get("Cache-Control") != "public, max-age=31536000" {
			t.Errorf("expected Cache-Control 'public, max-age=31536000', got %q", resp.Header.Get("Cache-Control"))
		}

		if gen.lastData != "https://example.com/match/1" {
			t.Errorf("expected generator data 'https://example.com/match/1', got %q", gen.lastData)
		}

		if gen.lastSize != 250 {
			t.Errorf("expected generator size 250, got %d", gen.lastSize)
		}

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "png-qr-content" {
			t.Errorf("expected body 'png-qr-content', got %q", string(body))
		}
	})

	t.Run("Custom valid size parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/qr?data=test&size=500", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		if gen.lastSize != 500 {
			t.Errorf("expected generator size 500, got %d", gen.lastSize)
		}
	})

	t.Run("Invalid size parameters fallback to default size 250", func(t *testing.T) {
		invalidSizes := []string{"abc", "-50", "0", "2000"}
		for _, sz := range invalidSizes {
			req := httptest.NewRequest(http.MethodGet, "/qr?data=test&size="+sz, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error for size %s: %v", sz, err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200 for size %s, got %d", sz, resp.StatusCode)
			}
			if gen.lastSize != 250 {
				t.Errorf("expected fallback size 250 for input %s, got %d", sz, gen.lastSize)
			}
		}
	})

	t.Run("Generator error returns 500", func(t *testing.T) {
		errGen := &mockQRGenerator{err: errors.New("qr generation error")}
		errHandler := handler.NewQRHandler(errGen)
		errApp := fiber.New()
		errApp.Get("/qr", errHandler.Generate)

		req := httptest.NewRequest(http.MethodGet, "/qr?data=fail", nil)
		resp, err := errApp.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "failed to generate QR code" {
			t.Errorf("expected body 'failed to generate QR code', got %q", string(body))
		}
	})
}
