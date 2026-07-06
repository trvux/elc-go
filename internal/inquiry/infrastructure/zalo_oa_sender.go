package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

// zaloMessageCSEndpoint is the Official Account "consultation message"
// endpoint — allowed for followers who've interacted with the OA recently
// (the "staff follows OA + sends it one message" bootstrap, see
// internal/inquiry/domain/zalo_follower.go). Confirmed against current
// developers.zalo.me docs (POST, header "access_token", JSON body).
const zaloMessageCSEndpoint = "https://openapi.zalo.me/v3.0/oa/message/cs"

// ZaloOASender implements domain.LeadNotifier by pushing a text message to
// every active Zalo OA follower (i.e. staff who opted in). A failure here
// must never surface as an error to the caller — see the package doc on
// domain.LeadNotifier — so every error path in NotifyNewLead is logged and
// swallowed.
type ZaloOASender struct {
	tokenRepo    domain.ZaloTokenRepository
	followerRepo domain.ZaloFollowerRepository
	adminBaseURL string
	log          *zap.Logger
	httpClient   *http.Client
}

var _ domain.LeadNotifier = (*ZaloOASender)(nil)

func NewZaloOASender(tokenRepo domain.ZaloTokenRepository, followerRepo domain.ZaloFollowerRepository, adminBaseURL string, log *zap.Logger) *ZaloOASender {
	return &ZaloOASender{
		tokenRepo:    tokenRepo,
		followerRepo: followerRepo,
		adminBaseURL: adminBaseURL,
		log:          log,
		httpClient:   &http.Client{},
	}
}

func (s *ZaloOASender) NotifyNewLead(ctx context.Context, inquiry *domain.Inquiry) error {
	token, err := s.tokenRepo.GetCurrent(ctx)
	if err != nil {
		s.log.Error("zalo oa: failed to load current token", zap.Error(err))
		return nil
	}
	if token == nil {
		s.log.Warn("zalo oa: no token bootstrapped yet (run cmd/zalo-authorize) — skipping notification")
		return nil
	}
	if token.NeedsRefresh(0) {
		// The background refresher (see zalo_token_refresher.go) should have
		// kept this fresh; if it's still stale, don't guess — skip and let
		// the refresher's own logging surface the real problem.
		s.log.Warn("zalo oa: access token expired, skipping notification until refresher catches up")
		return nil
	}

	followers, err := s.followerRepo.GetAllActive(ctx)
	if err != nil {
		s.log.Error("zalo oa: failed to load active followers", zap.Error(err))
		return nil
	}
	if len(followers) == 0 {
		s.log.Warn("zalo oa: no active followers yet — staff must follow the OA and message it once")
		return nil
	}

	text := buildLeadMessage(inquiry, s.adminBaseURL)
	for _, follower := range followers {
		if err := s.sendText(ctx, token.AccessToken(), follower.ZaloUserID(), text); err != nil {
			s.log.Error("zalo oa: failed to send lead notification",
				zap.String("zalo_user_id", follower.ZaloUserID()), zap.Error(err))
		}
	}
	return nil
}

func (s *ZaloOASender) sendText(ctx context.Context, accessToken, zaloUserID, text string) error {
	body, err := json.Marshal(map[string]any{
		"recipient": map[string]string{"user_id": zaloUserID},
		"message":   map[string]string{"text": text},
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, zaloMessageCSEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access_token", accessToken)

	res, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("zalo api returned status %d", res.StatusCode)
	}
	return nil
}

func buildLeadMessage(inquiry *domain.Inquiry, adminBaseURL string) string {
	interest := "Tư vấn chung"
	switch {
	case inquiry.ProductID() != nil:
		interest = "Sản phẩm (xem chi tiết trong admin)"
	case inquiry.ProjectID() != nil:
		interest = "Dự án (xem chi tiết trong admin)"
	case inquiry.ServiceID() != nil:
		interest = "Dịch vụ (xem chi tiết trong admin)"
	}

	return fmt.Sprintf(
		"Có khách hàng mới gửi yêu cầu tư vấn!\nHọ tên: %s\nSĐT: %s\nQuan tâm: %s\nXem chi tiết: %s/admin/inquiries",
		inquiry.Name(), inquiry.Phone(), interest, adminBaseURL,
	)
}
