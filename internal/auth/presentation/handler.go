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
	hasher      domain.PasswordHasher
	issuer      domain.TokenIssuer
	emailSender domain.EmailSender

	accessTokenTTL time.Duration
	// secureCookies must be true in production (HTTPS) — the Secure flag
	// makes browsers refuse to ever send the cookie over plain HTTP.
	secureCookies bool

	loginLimiter          *ratelimit.Limiter
	forgotPasswordLimiter *ratelimit.Limiter
	resetPasswordLimiter  *ratelimit.Limiter
	acceptInviteLimiter   *ratelimit.Limiter
}

func NewAuthHandler(
	userRepo domain.UserRepository,
	tokenRepo domain.VerificationTokenRepository,
	sessionRepo domain.SessionRepository,
	hasher domain.PasswordHasher,
	issuer domain.TokenIssuer,
	emailSender domain.EmailSender,
	accessTokenTTL time.Duration,
	secureCookies bool,
) *AuthHandler {
	return &AuthHandler{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		sessionRepo:    sessionRepo,
		hasher:         hasher,
		issuer:         issuer,
		emailSender:    emailSender,
		accessTokenTTL: accessTokenTTL,
		secureCookies:  secureCookies,
		// Limits are deliberately generous, not anti-abuse-grade — the goal
		// is to blunt naive brute force, not replace a WAF.
		loginLimiter:          ratelimit.New(10, 15*time.Minute),
		forgotPasswordLimiter: ratelimit.New(5, 15*time.Minute),
		resetPasswordLimiter:  ratelimit.New(10, 15*time.Minute),
		acceptInviteLimiter:   ratelimit.New(10, 15*time.Minute),
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if !h.loginLimiter.Allow(req.Identifier + "|" + httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("too many login attempts, try again later"))
		return
	}

	result, err := application.Login(r.Context(), h.userRepo, h.sessionRepo, h.hasher, h.issuer, application.LoginInput{
		Identifier: req.Identifier,
		Password:   req.Password,
		UserAgent:  r.UserAgent(),
		IPAddress:  httpserver.ClientIP(r),
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

const genericForgotPasswordMessage = "Nếu email tồn tại trong hệ thống, một liên kết đặt lại mật khẩu đã được gửi."

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	// Rate-limited silently: still returns the generic message, never a 429,
	// so a caller hammering this endpoint can't distinguish "rate limited"
	// from "email doesn't exist" from "email sent".
	if h.forgotPasswordLimiter.Allow(req.Email + "|" + httpserver.ClientIP(r)) {
		if err := application.ForgotPassword(r.Context(), h.userRepo, h.tokenRepo, h.emailSender, req.Email); err != nil {
			httpserver.WriteError(w, err)
			return
		}
	}

	httpserver.WriteJSON(w, http.StatusOK, messageResponse{Message: genericForgotPasswordMessage})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if !h.resetPasswordLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("too many attempts, try again later"))
		return
	}

	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if err := application.ResetPassword(r.Context(), h.userRepo, h.tokenRepo, h.sessionRepo, h.hasher, req.Token, req.Password); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, messageResponse{Message: "password has been reset"})
}

func (h *AuthHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	if !h.acceptInviteLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("too many attempts, try again later"))
		return
	}

	var req acceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	user, err := application.AcceptInvite(r.Context(), h.userRepo, h.tokenRepo, h.hasher, application.AcceptInviteInput{
		Token:    req.Token,
		Username: req.Username,
		Password: req.Password,
		Name:     req.Name,
		Phone:    req.Phone,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toUserResponse(user))
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

// CreateInvite requires RequireAuth + RequireRole(admin, super_admin) to
// have already run — this is the only way a new admin account can ever come
// into existence, so it must never be reachable anonymously.
func (h *AuthHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	userID, _ := httpserver.UserIDFromContext(r.Context())
	inviter, err := application.GetCurrentUser(r.Context(), h.userRepo, userID)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if inviter == nil {
		httpserver.WriteError(w, apperr.NewUnauthorizedError("user not found"))
		return
	}

	var req createInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	token, err := application.CreateInvite(r.Context(), h.userRepo, h.tokenRepo, h.emailSender, inviter, application.CreateInviteInput{
		Email: req.Email,
		Role:  domain.Role(req.Role),
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toInviteResponse(token))
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
