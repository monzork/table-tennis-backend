// Package storage uploads files to Supabase Storage over its REST API
// (plain net/http — Storage is just HTTPS, no SDK needed).
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SupabaseStorage uploads objects to a single Supabase Storage bucket.
type SupabaseStorage struct {
	baseURL string
	apiKey  string
	bucket  string
	client  *http.Client
}

func NewSupabaseStorage(baseURL, apiKey, bucket string) *SupabaseStorage {
	return &SupabaseStorage{baseURL: baseURL, apiKey: apiKey, bucket: bucket, client: http.DefaultClient}
}

// Upload PUTs data at path into the bucket (upserting on conflict) and
// returns the object's path within the bucket — not a URL. The bucket is
// expected to be private, so callers must go through SignedURL to view it.
func (s *SupabaseStorage) Upload(ctx context.Context, path string, data io.Reader, contentType string) (string, error) {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, s.bucket, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, data)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supabase storage upload failed: %s: %s", resp.Status, body)
	}

	return path, nil
}

// Download fetches an object's raw bytes directly from the bucket using the
// service API key (bypasses the need for a signed URL) — for server-side
// uses like embedding an ID photo into a generated PDF report.
func (s *SupabaseStorage) Download(ctx context.Context, path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("empty object path")
	}
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.baseURL, s.bucket, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("supabase storage download failed: %s: %s", resp.Status, body)
	}
	return io.ReadAll(resp.Body)
}

// SignedURL returns a temporary URL (valid for expiresInSeconds) that can
// view the private object at path without a service key. Callers must gate
// who gets to call this — anyone holding the URL can view the file until it
// expires.
func (s *SupabaseStorage) SignedURL(ctx context.Context, path string, expiresInSeconds int) (string, error) {
	if path == "" {
		return "", nil
	}
	url := fmt.Sprintf("%s/storage/v1/object/sign/%s/%s", s.baseURL, s.bucket, path)
	body := fmt.Sprintf(`{"expiresIn":%d}`, expiresInSeconds)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supabase storage sign failed: %s: %s", resp.Status, b)
	}

	var out struct {
		SignedURL string `json:"signedURL"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return s.baseURL + "/storage/v1" + out.SignedURL, nil
}
