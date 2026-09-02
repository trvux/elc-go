package domain

import "context"

// GoogleUserInfo is the verified identity Google hands back for an
// authorization code — Sub is the stable, unique Google account ID (never
// reused, unlike email which a user can change on their Google account).
type GoogleUserInfo struct {
	Sub     string
	Email   string
	Name    string
	Picture string
}

// GoogleAuthenticator abstracts exchanging an authorization code for a
// verified user identity, so the application layer doesn't depend directly
// on infrastructure/google (same interface-in-domain,
// implementation-in-infrastructure shape as EmailSender/TokenIssuer).
type GoogleAuthenticator interface {
	// Exchange trades an authorization code (from
	// google.accounts.oauth2.initCodeClient's popup flow) for the signed-in
	// user's info. redirectURI must match the origin of the page that opened
	// the popup — Google Identity Services' popup flow uses the calling
	// page's own origin as redirect_uri, not a fixed value, so the frontend
	// must supply it per request.
	Exchange(ctx context.Context, code string, redirectURI string) (*GoogleUserInfo, error)
}
