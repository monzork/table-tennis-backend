// Package oauth is a pure adapter around Google's OAuth2 + userinfo
// endpoints. It has no domain/application imports, same level as
// identity/, security/, qrcode/.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// defaultUserInfoURL is Google's OpenID-Connect userinfo endpoint. Calling
// it with the OAuth access token avoids needing JWT-signature verification
// (no coreos/go-oidc dependency needed for this single-provider case).
const defaultUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"

// GoogleUserInfo is the subset of Google's userinfo response we care about.
type GoogleUserInfo struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// GoogleClient wraps the Google OAuth2 "Sign in with Google" flow: building
// the consent-screen URL, then exchanging the returned code for a token and
// resolving that token to profile info.
type GoogleClient struct {
	cfg         *oauth2.Config
	userInfoURL string
	httpClient  *http.Client
}

// NewGoogleClient builds a GoogleClient configured for the "openid email
// profile" scopes against Google's standard OAuth2 endpoints.
func NewGoogleClient(clientID, clientSecret, redirectURL string) *GoogleClient {
	return &GoogleClient{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
		userInfoURL: defaultUserInfoURL,
		httpClient:  http.DefaultClient,
	}
}

// AuthCodeURL builds the URL to redirect the browser to for the Google
// consent screen, embedding the given CSRF state.
func (g *GoogleClient) AuthCodeURL(state string) string {
	return g.cfg.AuthCodeURL(state)
}

// Exchange trades an authorization code for an access token, then resolves
// that token against the userinfo endpoint to get the signed-in user's
// Google identity.
func (g *GoogleClient) Exchange(ctx context.Context, code string) (*GoogleUserInfo, error) {
	token, err := g.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned status %d", resp.StatusCode)
	}

	var info GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}
	return &info, nil
}
