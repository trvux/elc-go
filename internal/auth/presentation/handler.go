package presentation

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/auth/application"
	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/ratelimit"
)

const refreshCookieName = "refresh_token"

// AuthHandler is the composition root for the auth module: the only place
// allowed to hold concrete repository/service implementations and wire them
// into each application use case.
type AuthHandler struct {
	userRepo    domain.UserRepository
	tokenRepo   domain.VerificationTokenRepository
	sessionRepo domain.SessionRepository
	googleAuth  domain.GoogleAuthenticator
	issuer      domain.TokenIssuer
	emailSender domain.EmailSender
	adminRoles map[string]domain.Role

	accessTokenTTL time.Duration
	// secureCookies must be true in production (HTTPS) — the Secure flag
	// makes browsers refuse to ever send the cookie over plain HTTP.
	secureCookies bool

	googleLoginLimiter     *ratelimit.Limiter
	magicLinkLimiter       *ratelimit.Limiter
	verifyMagicLinkLimiter *ratelimit.Limiter
}

func NewAuthHandler(
	userRepo domain.UserRepository,
	tokenRepo domain.VerificationTokenRepository,
	sessionRepo domain.SessionRepository,
	googleAuth domain.GoogleAuthenticator,
	issuer domain.TokenIssuer,
	emailSender domain.EmailSender,
	adminRoles map[string]domain.Role,
	accessTokenTTL time.Duration,
	secureCookies bool,
) *AuthHandler {
	return &AuthHandler{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		sessionRepo:    sessionRepo,
		googleAuth:     googleAuth,
		issuer:         issuer,
		emailSender:    emailSender,
		adminRoles:     adminRoles,
		accessTokenTTL: accessTokenTTL,
		secureCookies:  secureCookies,
		// Limits are deliberately generous, not anti-abuse-grade — the goal
		// is to blunt naive brute force, not replace a WAF.
		googleLoginLimiter: ratelimit.New(20, 15*time.Minute),
		magicLinkLimiter:   ratelimit.New(5, 15*time.Minute),
		// verifyMagicLinkLimiter is the real defense against brute-forcing
		// the 6-digit code (10^6 combinations, MagicLinkTTL-minute window) —
		// keyed by email in HandleVerifyMagicLink, so it caps guesses against
		// one target regardless of how many different tokens/IPs an attacker
		// cycles through. See also application.magicLinkMaxAttempts, which
		// locks the token itself out server-side after too many wrong
		// guesses, independent of this in-memory limiter.
		verifyMagicLinkLimiter: ratelimit.New(5, application.MagicLinkTTL),
	}
}

// HandleGoogleLogin exchanges the authorization code from
// google.accounts.oauth2.initCodeClient's popup flow (see
// domain.GoogleAuthenticator) for a verified identity and issues a session.
func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req googleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if !h.googleLoginLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("too many attempts, try again later"))
		return
	}

	result, err := application.GoogleLogin(r.Context(), h.userRepo, h.sessionRepo, h.issuer, h.googleAuth, h.adminRoles, application.GoogleLoginInput{
		Code:      req.Code,
		UserAgent: r.UserAgent(),
		IPAddress: httpserver.ClientIP(r),
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	httpserver.WriteJSON(w, http.StatusOK, accessTokenResponse{
		User:         toUserResponse(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    int(h.accessTokenTTL.Seconds()),
	})
}

// HandleRequestMagicLink emails a sign-in link + 6-digit code for the given
// address — never reveals whether an account already exists for it (see
// application.RequestMagicLink's doc comment), always 204 unless rate
// limited or a real infrastructure error occurs.
func (h *AuthHandler) HandleRequestMagicLink(w http.ResponseWriter, r *http.Request) {
	var req requestMagicLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if !h.magicLinkLimiter.Allow(req.Email + "|" + httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("too many requests, try again later"))
		return
	}

	if err := application.RequestMagicLink(r.Context(), h.tokenRepo, h.emailSender, req.Email); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleVerifyMagicLink redeems either the link token or the email+code pair
// (see verifyMagicLinkRequest) and issues a session.
func (h *AuthHandler) HandleVerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	var req verifyMagicLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	userAgent, ip := r.UserAgent(), httpserver.ClientIP(r)

	var (
		result *application.LoginResult
		err    error
	)
	switch {
	case req.Token != "":
		// Rate-limited by the token itself: it's 32 random bytes, not
		// guessable, so this only blunts a misbehaving client retrying in a
		// loop — not a brute-force target like the code path below.
		if !h.verifyMagicLinkLimiter.Allow(req.Token) {
			httpserver.WriteError(w, apperr.NewTooManyRequestsError("too many attempts, try again later"))
			return
		}
		result, err = application.VerifyMagicLinkByToken(r.Context(), h.userRepo, h.sessionRepo, h.tokenRepo, h.issuer, h.adminRoles, req.Token, userAgent, ip)
	case req.Email != "" && req.Code != "":
		// Rate-limited by email — the actual defense against brute-forcing
		// the 6-digit code, independent of how many tokens/IPs an attacker
		// cycles through (see verifyMagicLinkLimiter's doc comment above).
		if !h.verifyMagicLinkLimiter.Allow(req.Email) {
			httpserver.WriteError(w, apperr.NewTooManyRequestsError("too many attempts, try again later"))
			return
		}
		result, err = application.VerifyMagicLinkByCode(r.Context(), h.userRepo, h.sessionRepo, h.tokenRepo, h.issuer, h.adminRoles, req.Email, req.Code, userAgent, ip)
	default:
		httpserver.WriteError(w, apperr.NewValidationError("token or (email, code) required", nil))
		return
	}
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	httpserver.WriteJSON(w, http.StatusOK, accessTokenResponse{
		User:         toUserResponse(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    int(h.accessTokenTTL.Seconds()),
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var raw string
	if cookie, err := r.Cookie(refreshCookieName); err == nil {
		raw = cookie.Value
	}

	if err := application.Logout(r.Context(), h.sessionRepo, raw); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	h.clearRefreshCookie(w)
	httpserver.WriteJSON(w, http.StatusOK, messageResponse{Message: "logged out"})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		httpserver.WriteError(w, apperr.NewUnauthorizedError("missing refresh token"))
		return
	}

	result, err := application.RefreshToken(r.Context(), h.userRepo, h.sessionRepo, h.issuer, cookie.Value)
	if err != nil {
		h.clearRefreshCookie(w)
		httpserver.WriteError(w, err)
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	httpserver.WriteJSON(w, http.StatusOK, accessTokenResponse{
		User:         toUserResponse(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    int(h.accessTokenTTL.Seconds()),
	})
}

// Me requires RequireAuth to have already run (see routes.go), so the user
// ID in context has already been through JWT verification.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, _ := httpserver.UserIDFromContext(r.Context())

	user, err := application.GetCurrentUser(r.Context(), h.userRepo, userID)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if user == nil {
		httpserver.WriteError(w, apperr.NewUnauthorizedError("user not found"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toUserResponse(user))
}

// UpdateProfile requires RequireAuth only (see routes.go) — every account is
// always allowed to edit its own name/email/avatar, no permission check
// beyond "is this a real logged-in user".
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	actorID, _ := httpserver.UserIDFromContext(r.Context())
	actor, err := application.GetCurrentUser(r.Context(), h.userRepo, actorID)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if actor == nil {
		httpserver.WriteError(w, apperr.NewUnauthorizedError("user not found"))
		return
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	updated, err := application.UpdateProfile(r.Context(), h.userRepo, actor, application.UpdateProfileInput{
		Name:      req.Name,
		Email:     req.Email,
		AvatarURL: req.AvatarURL,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toUserResponse(updated))
}

// ListUsers requires RequireAuth + RequirePermission(users:manage) — see
// routes.go. Backs the admin-panel user management screen.
func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := application.ListUsers(r.Context(), h.userRepo)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toUserResponseList(users))
}

// UpdateUser requires RequireAuth + RequirePermission(users:manage). The
// actual privilege-escalation / self-management guards live in
// application.UpdateUser — this handler only decodes the request and
// resolves who the caller is.
func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	actorID, _ := httpserver.UserIDFromContext(r.Context())
	actor, err := application.GetCurrentUser(r.Context(), h.userRepo, actorID)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if actor == nil {
		httpserver.WriteError(w, apperr.NewUnauthorizedError("user not found"))
		return
	}

	targetID := chi.URLParam(r, "id")

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := application.UpdateUserInput{}
	if req.Role != nil {
		role := domain.Role(*req.Role)
		input.Role = &role
	}
	if req.Status != nil {
		status := domain.UserStatus(*req.Status)
		input.Status = &status
	}

	updated, err := application.UpdateUser(r.Context(), h.userRepo, h.sessionRepo, actor, targetID, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toUserResponse(updated))
}

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, rawToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    rawToken,
		Path:     "/auth",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(application.RefreshTokenTTL.Seconds()),
	})
}

func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/auth",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
