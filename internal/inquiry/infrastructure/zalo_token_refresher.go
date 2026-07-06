package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

// zaloOAuthTokenEndpoint issues/refreshes OAuth tokens for a Zalo App.
// Confirmed against current developers.zalo.me docs: access_token lives 1
// hour, refresh_token lives 30 days AND is single-use — every refresh
// response contains a brand new refresh_token that must be persisted, the
// old one stops working immediately.
const zaloOAuthTokenEndpoint = "https://oauth.zaloapp.com/v4/access_token"

// refreshSafetyMargin refreshes well before the 1-hour access token expiry
// so a slow ticker tick or a transient Zalo outage never leaves a request
// going out with an expired token.
const refreshSafetyMargin = 15 * time.Minute

type zaloTokenResponse struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	ExpiresIn             string `json:"expires_in"`
	RefreshTokenExpiresIn string `json:"refresh_token_expires_in"`
	Error                 int    `json:"error"`
	ErrorName             string `json:"error_name"`
	ErrorReason           string `json:"error_reason"`
}

// ZaloTokenRefresher keeps the singleton zalo_oa_tokens row fresh via a
// background goroutine — this codebase has no prior background-task
// pattern (confirmed before writing this), so it's a new, narrowly-scoped
// one: a single ticker calling refreshIfNeeded, nothing generic.
type ZaloTokenRefresher struct {
	tokenRepo  domain.ZaloTokenRepository
	appID      string
	appSecret  string
	log        *zap.Logger
	interval   time.Duration
	httpClient *http.Client
}

func NewZaloTokenRefresher(tokenRepo domain.ZaloTokenRepository, appID, appSecret string, log *zap.Logger, interval time.Duration) *ZaloTokenRefresher {
	return &ZaloTokenRefresher{
		tokenRepo:  tokenRepo,
		appID:      appID,
		appSecret:  appSecret,
		log:        log,
		interval:   interval,
		httpClient: &http.Client{},
	}
}

// Start blocks until ctx is cancelled — run it in its own goroutine.
func (r *ZaloTokenRefresher) Start(ctx context.Context) {
	r.refreshIfNeeded(ctx) // catch up immediately on startup, don't wait a full interval

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.refreshIfNeeded(ctx)
		}
	}
}

func (r *ZaloTokenRefresher) refreshIfNeeded(ctx context.Context) {
	current, err := r.tokenRepo.GetCurrent(ctx)
	if err != nil {
		r.log.Error("zalo token refresher: failed to load current token", zap.Error(err))
		return
	}
	if current == nil {
		// Not bootstrapped yet (cmd/zalo-authorize hasn't been run) —
		// nothing to refresh, this is expected until the OA is set up.
		return
	}
	if !current.NeedsRefresh(refreshSafetyMargin) {
		return
	}

	form := url.Values{}
	form.Set("refresh_token", current.RefreshToken())
	form.Set("app_id", r.appID)
	form.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, zaloOAuthTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		r.log.Error("zalo token refresher: failed to build request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("secret_key", r.appSecret)

	res, err := r.httpClient.Do(req)
	if err != nil {
		r.log.Error("zalo token refresher: request failed", zap.Error(err))
		return
	}
	defer res.Body.Close()

	var parsed zaloTokenResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		r.log.Error("zalo token refresher: failed to decode response", zap.Error(err))
		return
	}
	if parsed.AccessToken == "" || parsed.RefreshToken == "" {
		r.log.Error("zalo token refresher: refresh failed — refresh_token may have expired (30-day lifetime), re-run cmd/zalo-authorize",
			zap.Int("error", parsed.Error), zap.String("error_name", parsed.ErrorName), zap.String("error_reason", parsed.ErrorReason))
		return
	}

	expiresInSeconds, err := strconv.Atoi(parsed.ExpiresIn)
	if err != nil {
		r.log.Error("zalo token refresher: unexpected expires_in value", zap.String("expires_in", parsed.ExpiresIn))
		return
	}

	newToken := domain.NewZaloOAToken(parsed.AccessToken, parsed.RefreshToken, time.Now().Add(time.Duration(expiresInSeconds)*time.Second))
	if err := r.tokenRepo.Save(ctx, newToken); err != nil {
		r.log.Error("zalo token refresher: failed to save refreshed token", zap.Error(err))
		return
	}
	r.log.Info("zalo token refresher: refreshed access token", zap.Time("expires_at", newToken.ExpiresAt()))
}
