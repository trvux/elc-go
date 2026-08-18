package presentation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/trvux/elc-go/internal/ai/application"
	"github.com/trvux/elc-go/internal/ai/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/ratelimit"
)

// chatTimeout bounds one round trip through the classifier + streaming chat
// completion (which may itself make a few sequential calls: tool rounds
// within a model, and another model on fallback) — every external call
// needs a timeout, and a hung upstream must not hold an HTTP handler
// goroutine open indefinitely.
const chatTimeout = 60 * time.Second

// blockedReplyVI is what the customer sees when the guardrail classifier
// rejects a message — no chat model is ever called for a blocked turn (the
// whole cost-saving point of running the classifier first).
const blockedReplyVI = "Xin lỗi, mình chỉ có thể tư vấn về sản phẩm và dịch vụ của ELC thôi ạ. Anh/chị cần hỗ trợ gì về sản phẩm điện máy không?"

var (
	errVisitorIDMissing      = errors.New("ai: visitor_id missing from context — EnsureVisitorID must be mounted on this route")
	errNoChatModelConfigured = errors.New("ai: no active chat model configured")
	errStreamingUnsupported  = errors.New("ai: response writer does not support streaming")
)

// ChatRateLimit/ChatRateLimitWindow: 20 messages per IP per 10 minutes —
// generous for a real back-and-forth with a customer, tight enough to
// blunt a script hammering a paid LLM API. Exported so cmd/server/main.go
// (the composition root that now picks in-memory vs. Redis-backed, see
// docs/rfc/2026-08-18-ai-chat-redis.md) builds whichever ratelimit.
// RateLimiter it chooses with the same numbers.
const (
	ChatRateLimit       = 20
	ChatRateLimitWindow = 10 * time.Minute
)

// AIHandler is the composition root for the AI chat module.
type AIHandler struct {
	clientFactory    domain.LLMClientFactory
	modelRepo        domain.ModelRepository
	conversationRepo domain.ConversationRepository
	tools            []application.Tool
	// cache is optional (nil = disabled, see domain.Cache's doc comment) —
	// backs ClassifyMessage's/search_products' result caching.
	cache       domain.Cache
	chatLimiter ratelimit.RateLimiter
}

func NewAIHandler(
	clientFactory domain.LLMClientFactory,
	modelRepo domain.ModelRepository,
	conversationRepo domain.ConversationRepository,
	tools []application.Tool,
	cache domain.Cache,
	chatLimiter ratelimit.RateLimiter,
) *AIHandler {
	return &AIHandler{
		clientFactory:    clientFactory,
		modelRepo:        modelRepo,
		conversationRepo: conversationRepo,
		tools:            tools,
		cache:            cache,
		chatLimiter:      chatLimiter,
	}
}

// Chat handles a public chat turn over Server-Sent Events: one new customer
// message in, the assistant's reply streamed out delta by delta as the
// model generates it. The conversation is persisted server-side (keyed by
// the visitor_id cookie EnsureVisitorID attaches, see routes.go), so the
// caller only ever sends the newest message — the server owns history.
//
// Every early-exit before streaming starts (rate limit, validation, a
// config problem) is a normal JSON error response via httpserver.WriteError.
// Once the SSE response has been opened — right before the canned refusal
// or the real model stream — nothing can fall back to a plain HTTP error
// anymore; see sse.go's sendError.
func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if !h.chatLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}

	visitorID, ok := httpserver.VisitorIDFromContext(r.Context())
	if !ok {
		httpserver.WriteError(w, apperr.NewInternalError(errVisitorIDMissing))
		return
	}
	var userID *string
	if uid, ok := httpserver.UserIDFromContext(r.Context()); ok && uid != "" {
		userID = &uid
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}
	if len(req.Message) == 0 || len(req.Message) > maxMessageLength {
		httpserver.WriteError(w, apperr.NewValidationError("message length is invalid", nil))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), chatTimeout)
	defer cancel()

	conv, err := h.conversationRepo.GetOrCreateByVisitor(ctx, visitorID, userID)
	if err != nil {
		httpserver.WriteError(w, apperr.NewInternalError(err))
		return
	}

	if _, err := h.conversationRepo.AppendMessage(ctx, &domain.ConversationMessage{
		ConversationID: conv.ID,
		Role:           domain.RoleUser,
		Content:        req.Message,
	}); err != nil {
		httpserver.WriteError(w, apperr.NewInternalError(err))
		return
	}

	classifierModels, err := application.ResolveActiveModels(ctx, h.modelRepo, domain.ModelRoleClassifier)
	if err != nil {
		httpserver.WriteError(w, apperr.NewInternalError(err))
		return
	}

	if verdict := application.ClassifyMessage(ctx, h.clientFactory, classifierModels, h.cache, req.Message); verdict != nil && !verdict.OnTopic {
		stream, ok := newSSEWriter(w)
		if !ok {
			httpserver.WriteError(w, apperr.NewInternalError(errStreamingUnsupported))
			return
		}
		stream.sendDelta(blockedReplyVI)
		stream.sendDone(true)

		reason := verdict.Reason
		// Best-effort: the SSE response has already succeeded from the
		// client's perspective, so a persistence failure here has nowhere
		// left to be reported — same trade-off as the success path below.
		_, _ = h.conversationRepo.AppendMessage(ctx, &domain.ConversationMessage{
			ConversationID: conv.ID,
			Role:           domain.RoleAssistant,
			Content:        blockedReplyVI,
			BlockedReason:  &reason,
		})
		return
	}

	chatModels, err := application.ResolveActiveModels(ctx, h.modelRepo, domain.ModelRoleChat)
	if err != nil {
		httpserver.WriteError(w, apperr.NewInternalError(err))
		return
	}
	if len(chatModels) == 0 {
		httpserver.WriteError(w, apperr.NewInternalError(errNoChatModelConfigured))
		return
	}

	history, err := h.conversationRepo.ListRecentMessages(ctx, conv.ID, chatHistoryLimit)
	if err != nil {
		httpserver.WriteError(w, apperr.NewInternalError(err))
		return
	}

	stream, ok := newSSEWriter(w)
	if !ok {
		httpserver.WriteError(w, apperr.NewInternalError(errStreamingUnsupported))
		return
	}

	outcome, err := application.SendChatMessageStream(ctx, h.clientFactory, chatModels, h.tools, toDomainMessages(history), stream.sendDelta)
	if err != nil {
		// Headers are already sent — this is the only way left to signal
		// failure to the client, see sse.go's sendError. outcome is still
		// non-nil whenever some text already reached the client before the
		// failure (see SendChatMessageStream's doc comment) — persist that
		// partial reply marked incomplete instead of losing it, per the
		// Phase 2 RFC §2.4.
		stream.sendError()
		if outcome != nil {
			h.persistAssistantMessage(ctx, conv.ID, outcome, true)
		}
		return
	}

	h.persistAssistantMessage(ctx, conv.ID, outcome, false)
	stream.sendDone(false)
}

// persistAssistantMessage is best-effort: by the time it's called the SSE
// response has already succeeded (or failed) from the client's point of
// view, so a persistence error here has nowhere left to be reported.
func (h *AIHandler) persistAssistantMessage(ctx context.Context, conversationID string, outcome *application.ChatOutcome, incomplete bool) {
	cost := outcome.Model.Pricing.Cost(outcome.Usage, time.Now())
	_, _ = h.conversationRepo.AppendMessage(ctx, &domain.ConversationMessage{
		ConversationID: conversationID,
		Role:           domain.RoleAssistant,
		Content:        outcome.Message.Content,
		Incomplete:     incomplete,
		ProviderID:     &outcome.Model.ProviderID,
		ModelID:        &outcome.Model.ModelID,
		Usage:          &outcome.Usage,
		CostUSD:        &cost,
		ProductsShown:  outcome.ProductsShown,
	})
}

func toDomainMessages(messages []*domain.ConversationMessage) []domain.Message {
	out := make([]domain.Message, len(messages))
	for i, m := range messages {
		out[i] = domain.Message{Role: m.Role, Content: m.Content}
	}
	return out
}
