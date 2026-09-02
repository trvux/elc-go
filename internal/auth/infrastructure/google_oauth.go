package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/trvux/elc-go/internal/auth/domain"
)

// GoogleOAuthClient implements domain.GoogleAuthenticator against Google's
// real OAuth2 + userinfo endpoints.
type GoogleOAuthClient struct {
	clientID     string
	clientSecret string
}

var _ domain.GoogleAuthenticator = (*GoogleOAuthClient)(nil)

func NewGoogleOAuthClient(clientID, clientSecret string) *GoogleOAuthClient {
	return &GoogleOAuthClient{clientID: clientID, clientSecret: clientSecret}
}

type googleUserInfoResponse struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// Exchange trades an authorization code for an access token, then calls
// Google's userinfo endpoint to get the verified identity behind it.
// redirectURI must match the origin of the page that requested the code (see
// domain.GoogleAuthenticator's doc comment).
func (c *GoogleOAuthClient) Exchange(ctx context.Context, code string, redirectURI string) (*domain.GoogleUserInfo, error) {
	config := &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     oauth2.Endpoint{TokenURL: "https://oauth2.googleapis.com/token"},
	}

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange google code: %w", err)
	}

	client := config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("fetch google userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo returned status %d", resp.StatusCode)
	}

	var info googleUserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode google userinfo: %w", err)
	}

	// Only Google-verified emails prove inbox ownership — an unverified email
	// on a Google account can be set to anything, so trusting it here would
	// let someone claim any address without ever receiving mail there.
	if !info.EmailVerified {
		return nil, fmt.Errorf("google account email is not verified")
	}

	return &domain.GoogleUserInfo{Sub: info.Sub, Email: info.Email, Name: info.Name, Picture: info.Picture}, nil
}
