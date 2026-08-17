package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestGoogleClient builds a GoogleClient wired to httptest stand-ins for
// both the token endpoint and the userinfo endpoint, so Exchange can be
// tested without hitting real Google infra.
func newTestGoogleClient(t *testing.T, tokenHandler, userInfoHandler http.HandlerFunc) (*GoogleClient, *httptest.Server, *httptest.Server) {
	t.Helper()

	tokenSrv := httptest.NewServer(tokenHandler)
	t.Cleanup(tokenSrv.Close)

	userInfoSrv := httptest.NewServer(userInfoHandler)
	t.Cleanup(userInfoSrv.Close)

	g := NewGoogleClient("client-id", "client-secret", "http://localhost/callback")
	g.cfg.Endpoint.TokenURL = tokenSrv.URL
	g.cfg.Endpoint.AuthURL = tokenSrv.URL + "/auth"
	g.userInfoURL = userInfoSrv.URL

	return g, tokenSrv, userInfoSrv
}

func tokenOKHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"access_token":"test-access-token","token_type":"Bearer","expires_in":3600}`))
}

func TestNewGoogleClient(t *testing.T) {
	g := NewGoogleClient("cid", "csecret", "http://localhost/cb")
	if g.cfg.ClientID != "cid" || g.cfg.ClientSecret != "csecret" || g.cfg.RedirectURL != "http://localhost/cb" {
		t.Fatalf("unexpected config: %+v", g.cfg)
	}
	if g.userInfoURL != defaultUserInfoURL {
		t.Fatalf("expected default userinfo URL, got %q", g.userInfoURL)
	}
}

func TestGoogleClient_AuthCodeURL(t *testing.T) {
	g := NewGoogleClient("cid", "csecret", "http://localhost/cb")
	url := g.AuthCodeURL("state123")
	if !strings.Contains(url, "state123") {
		t.Fatalf("expected state in URL, got %q", url)
	}
	if !strings.Contains(url, "cid") {
		t.Fatalf("expected client id in URL, got %q", url)
	}
}

func TestGoogleClient_Exchange_Success(t *testing.T) {
	g, _, _ := newTestGoogleClient(t, tokenOKHandler, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token" {
			t.Errorf("expected bearer token header, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GoogleUserInfo{
			Sub:     "sub-123",
			Email:   "user@example.com",
			Name:    "Test User",
			Picture: "http://pic",
		})
	})

	info, err := g.Exchange(context.Background(), "auth-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Sub != "sub-123" || info.Email != "user@example.com" || info.Name != "Test User" || info.Picture != "http://pic" {
		t.Fatalf("unexpected userinfo: %+v", info)
	}
}

func TestGoogleClient_Exchange_TokenError(t *testing.T) {
	g, _, _ := newTestGoogleClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("userinfo endpoint should not be called when token exchange fails")
	})

	if _, err := g.Exchange(context.Background(), "bad-code"); err == nil {
		t.Fatal("expected error from failed token exchange, got nil")
	}
}

func TestGoogleClient_Exchange_UserInfoNon200(t *testing.T) {
	g, _, _ := newTestGoogleClient(t, tokenOKHandler, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	if _, err := g.Exchange(context.Background(), "auth-code"); err == nil {
		t.Fatal("expected error from non-200 userinfo response, got nil")
	}
}

func TestGoogleClient_Exchange_UserInfoBadJSON(t *testing.T) {
	g, _, _ := newTestGoogleClient(t, tokenOKHandler, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	})

	if _, err := g.Exchange(context.Background(), "auth-code"); err == nil {
		t.Fatal("expected error from malformed userinfo JSON, got nil")
	}
}
