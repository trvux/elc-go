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

var (
	errVisitorIDMissing      = errors.New("ai: visitor_id missing from context — EnsureVisitorID must be mounted on this route")
	errNoChatModelConfigured = errors.New("ai: no active chat model configured")
)

// chatTimeout bounds one round trip through the classifier + SendChatMessage
// (which may itself make a few sequential calls: tool rounds within a
// model, and another model on fallback) — every external call needs a
// timeout, and a hung upstream must not hold an HTTP handler goroutine open
// indefinitely.
const chatTimeout = 30 * time.Second

// blockedReplyVI is what the customer sees when the guardrail classifier
// rejects a message — no chat model is ever called for a blocked turn (the
// whole cost-saving point of running the classifier first).
const blockedReplyVI = "Xin lỗi, mình chỉ có thể tư vấn về sản phẩm và dịch vụ của ELC thôi ạ. Anh/chị cần hỗ trợ gì về sản phẩm điện máy không?"

// AIHandler is the composition root for the AI chat module.
type AIHandler struct {
	clientFactory    domain.LLMClientFactory
	modelRepo        domain.ModelRepository
	conversationRepo domain.ConversationRepository
	tools            []application.Tool
	chatLimiter      *ratelimit.Limiter
}

func NewAIHandler(
	clientFactory domain.LLMClientFactory,
	modelRepo domain.ModelRepository,
	conversationRepo domain.ConversationRepository,
	tools []application.Tool,
) *AIHandler {
	return &AIHandler{
		clientFactory:    clientFactory,
		modelRepo:        modelRepo,
		conversationRepo: conversationRepo,
		tools:            tools,
		// 20 messages per IP per 10 minutes — generous for a real
		// back-and-forth with a customer, tight enough to blunt a script
		// hammering a paid LLM API.
		chatLimiter: ratelimit.New(20, 10*time.Minute),
	}
}

// Chat handles a public chat turn: one new customer message in, the
// assistant's next reply out. The conversation is persisted server-side
// (keyed by the visitor_id cookie EnsureVisitorID attaches, see routes.go),
// so the caller only ever sends the newest message — the server owns
// history, unlike v1's now-superseded stateless "resend everything"
// contract.
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
	if verdict := application.ClassifyMessage(ctx, h.clientFactory, classifierModels, req.Message); verdict != nil && !verdict.OnTopic {
		reason := verdict.Reason
		if _, err := h.conversationRepo.AppendMessage(ctx, &domain.ConversationMessage{
			ConversationID: conv.ID,
			Role:           domain.RoleAssistant,
			Content:        blockedReplyVI,
			BlockedReason:  &reason,
		}); err != nil {
			httpserver.WriteError(w, apperr.NewInternalError(err))
			return
		}
		httpserver.WriteJSON(w, http.StatusOK, chatResponse{Message: toChatMessageDTO(domain.RoleAssistant, blockedReplyVI), Blocked: true})
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

	outcome, err := application.SendChatMessage(ctx, h.clientFactory, chatModels, h.tools, toDomainMessages(history))
	if err != nil {
		httpserver.WriteError(w, apperr.NewInternalError(err))
		return
	}

	cost := outcome.Model.Pricing.Cost(outcome.Usage, time.Now())
	if _, err := h.conversationRepo.AppendMessage(ctx, &domain.ConversationMessage{
		ConversationID: conv.ID,
		Role:           domain.RoleAssistant,
		Content:        outcome.Message.Content,
		ProviderID:     &outcome.Model.ProviderID,
		ModelID:        &outcome.Model.ModelID,
		Usage:          &outcome.Usage,
		CostUSD:        &cost,
		ProductsShown:  outcome.ProductsShown,
	}); err != nil {
		httpserver.WriteError(w, apperr.NewInternalError(err))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, chatResponse{Message: toChatMessageDTO(domain.RoleAssistant, outcome.Message.Content)})
}

func toDomainMessages(messages []*domain.ConversationMessage) []domain.Message {
	out := make([]domain.Message, len(messages))
	for i, m := range messages {
		out[i] = domain.Message{Role: m.Role, Content: m.Content}
	}
	return out
}
