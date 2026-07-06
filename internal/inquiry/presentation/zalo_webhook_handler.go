package presentation

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/trvux/elc-go/internal/inquiry/application"
	"github.com/trvux/elc-go/internal/inquiry/domain"
	"github.com/trvux/elc-go/internal/inquiry/infrastructure"
)

// ZaloWebhookHandler receives Zalo OA events (follow/unfollow, messages from
// users) and turns them into ZaloOAFollower rows — see
// internal/inquiry/domain/zalo_follower.go for why this is the only way a
// staff member's Zalo identity gets captured (they follow the OA and
// message it once).
type ZaloWebhookHandler struct {
	followerRepo domain.ZaloFollowerRepository
	appSecret    string // empty if Zalo OA isn't configured yet
	log          *zap.Logger
}

func NewZaloWebhookHandler(followerRepo domain.ZaloFollowerRepository, appSecret string, log *zap.Logger) *ZaloWebhookHandler {
	return &ZaloWebhookHandler{followerRepo: followerRepo, appSecret: appSecret, log: log}
}

// Receive is always mounted (see routes.go) regardless of whether Zalo OA
// credentials are configured, so the webhook URL can be registered in
// Zalo's console at any time without a redeploy — it just no-ops with a
// warning until ZALO_OA_APP_SECRET is set.
func (h *ZaloWebhookHandler) Receive(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var evt zaloWebhookEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if h.appSecret == "" {
		h.log.Warn("zalo webhook received but ZALO_OA_APP_SECRET is not configured — ignoring",
			zap.String("event_name", evt.EventName))
		w.WriteHeader(http.StatusOK)
		return
	}

	signature := r.Header.Get("X-ZEvent-Signature")
	if !infrastructure.VerifyZaloWebhookSignature(evt.AppID, string(body), evt.Timestamp, h.appSecret, signature) {
		h.log.Warn("zalo webhook signature mismatch — rejecting",
			zap.String("event_name", evt.EventName),
			zap.String("debug_raw_body", string(body)),
			zap.String("debug_app_id", evt.AppID),
			zap.String("debug_timestamp", evt.Timestamp),
			zap.String("debug_signature_header", signature),
			zap.Any("debug_all_headers", r.Header))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	switch {
	case evt.EventName == "follow" && evt.Follower != nil:
		if err := application.RegisterZaloFollower(r.Context(), h.followerRepo, evt.Follower.ID, nil); err != nil {
			h.log.Error("zalo webhook: failed to register follower", zap.Error(err))
		}
	case evt.EventName == "unfollow" && evt.Follower != nil:
		if err := application.DeactivateZaloFollower(r.Context(), h.followerRepo, evt.Follower.ID); err != nil {
			h.log.Error("zalo webhook: failed to deactivate follower", zap.Error(err))
		}
	// A staff member messaging the OA also proves they can receive pushes,
	// same as following it — register them here too. Zalo's docs name the
	// anonymous variant explicitly (`anonymous_send_text`); the
	// non-anonymous one wasn't spelled out as precisely, so this matches
	// broadly on any user-originated "send" event rather than one exact
	// string — verify the real event_name against a live webhook payload
	// once Zalo OA is configured, and tighten this if needed.
	case evt.Sender != nil && strings.Contains(evt.EventName, "send") && !strings.HasPrefix(evt.EventName, "oa_"):
		if err := application.RegisterZaloFollower(r.Context(), h.followerRepo, evt.Sender.ID, nil); err != nil {
			h.log.Error("zalo webhook: failed to register follower from message event", zap.Error(err))
		}
	}

	w.WriteHeader(http.StatusOK)
}
