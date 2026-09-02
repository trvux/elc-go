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
	// user's info. There is no redirect_uri parameter here on purpose: GIS's
	// JS popup code-client flow always redeems against the literal string
	// "postmessage" — not the calling page's origin — regardless of what
	// domain the popup was opened from. Passing the page's own origin here
	// instead makes every exchange fail with a Google-side redirect_uri
	// mismatch, surfaced to the caller as "could not verify google account".
	Exchange(ctx context.Context, code string) (*GoogleUserInfo, error)
}
