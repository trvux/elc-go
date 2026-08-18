package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/trvux/elc-go/internal/ai/application"
	"github.com/trvux/elc-go/internal/ai/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/ratelimit"
)

// chatTimeout bounds one round trip through SendChatMessage (which may
// itself make up to a few sequential LLM calls for tool rounds) — every
// external call needs a timeout, and a hung upstream must not hold an HTTP
// handler goroutine open indefinitely.
const chatTimeout = 30 * time.Second

// AIHandler is the composition root for the AI chat module.
type AIHandler struct {
	client      domain.LLMClient
	tools       []application.Tool
	chatLimiter *ratelimit.Limiter
}

// NewAIHandler wires an AIHandler. tools is built by the caller (main.go),
// e.g. []application.Tool{ {Definition, Execute} } from
// infrastructure.NewProductSearchTool — kept as a parameter here rather than
// constructed inline so this package never needs to import the product
// module directly.
func NewAIHandler(client domain.LLMClient, tools []application.Tool) *AIHandler {
	return &AIHandler{
		client: client,
		tools:  tools,
		// 20 messages per IP per 10 minutes — generous for a real
		// back-and-forth with a customer, tight enough to blunt a script
		// hammering a paid LLM API.
		chatLimiter: ratelimit.New(20, 10*time.Minute),
	}
}

// Chat handles a public chat turn: the full conversation so far (messages)
// in, the assistant's next reply out. Stateless by design — no server-side
// conversation storage, the caller (frontend) resends history each request.
func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if !h.chatLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if len(req.Messages) == 0 {
		httpserver.WriteError(w, apperr.NewValidationError("messages must not be empty", nil))
		return
	}
	if len(req.Messages) > maxHistoryMessages {
		httpserver.WriteError(w, apperr.NewValidationError("too many messages", nil))
		return
	}
	for _, m := range req.Messages {
		if m.Role != string(domain.RoleUser) && m.Role != string(domain.RoleAssistant) {
			httpserver.WriteError(w, apperr.NewValidationError("message role must be \"user\" or \"assistant\"", nil))
			return
		}
		if len(m.Content) == 0 || len(m.Content) > maxMessageLength {
			httpserver.WriteError(w, apperr.NewValidationError("message content length is invalid", nil))
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), chatTimeout)
	defer cancel()

	reply, err := application.SendChatMessage(ctx, h.client, h.tools, toDomainMessages(req.Messages))
	if err != nil {
		httpserver.WriteError(w, apperr.NewInternalError(err))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toChatResponse(reply))
}
