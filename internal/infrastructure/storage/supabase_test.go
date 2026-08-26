package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSupabaseStorageUpload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/storage/v1/object/player-ids/id_front/abc.jpg" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing/wrong auth header: %s", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewSupabaseStorage(srv.URL, "test-key", "player-ids")
	path, err := s.Upload(context.Background(), "id_front/abc.jpg", strings.NewReader("fake-image"), "image/jpeg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "id_front/abc.jpg" {
		t.Errorf("got %q, want the object path unchanged", path)
	}
}

func TestSupabaseStorageSignedURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/storage/v1/object/sign/player-ids/id_front/abc.jpg" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing/wrong auth header: %s", r.Header.Get("Authorization"))
		}
		w.Write([]byte(`{"signedURL":"/object/sign/player-ids/id_front/abc.jpg?token=xyz"}`))
	}))
	defer srv.Close()

	s := NewSupabaseStorage(srv.URL, "test-key", "player-ids")
	url, err := s.SignedURL(context.Background(), "id_front/abc.jpg", 300)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := srv.URL + "/storage/v1/object/sign/player-ids/id_front/abc.jpg?token=xyz"
	if url != want {
		t.Errorf("got %q, want %q", url, want)
	}
}

func TestSupabaseStorageSignedURLEmptyPath(t *testing.T) {
	s := NewSupabaseStorage("https://example.supabase.co", "test-key", "player-ids")
	url, err := s.SignedURL(context.Background(), "", 300)
	if err != nil || url != "" {
		t.Errorf("expected empty result for empty path, got %q, %v", url, err)
	}
}

func TestSupabaseStorageUploadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	s := NewSupabaseStorage(srv.URL, "bad-key", "player-ids")
	if _, err := s.Upload(context.Background(), "x.jpg", strings.NewReader("x"), "image/jpeg"); err == nil {
		t.Fatal("expected error on non-2xx response")
	}
}
