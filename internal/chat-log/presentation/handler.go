package presentation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/trvux/elc-go/internal/chat-log/application"
	"github.com/trvux/elc-go/internal/chat-log/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/ratelimit"
)

type ChatLogHandler struct {
	repo         domain.ChatLogRepository
	writeLimiter *ratelimit.Limiter
}

func NewChatLogHandler(repo domain.ChatLogRepository) *ChatLogHandler {
	return &ChatLogHandler{
		repo: repo,
		// Generous — defends against a scripted abuse loop, not real
		// visitor usage (same reasoning as inquiry's/recently-viewed's own
		// write limiters).
		writeLimiter: ratelimit.New(120, 10*time.Minute),
	}
}

// Create logs one shopper message from the chat finder — public and
// anonymous, keyed by the visitor_id cookie EnsureVisitorID (mounted in
// routes.go) attached to the request. Fire-and-forget from the frontend's
// perspective (see ProductChatFinder.tsx): a failure here never blocks or
// alters the actual chat response, only the logging.
func (h *ChatLogHandler) Create(w http.ResponseWriter, r *http.Request) {
	id, ok := httpserver.VisitorIDFromContext(r.Context())
	if !ok || id == "" {
		httpserver.WriteError(w, apperr.NewInternalError(fmt.Errorf("visitor_id missing from context")))
		return
	}

	if !h.writeLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}

	var req createChatLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	entry, err := application.CreateChatLog(r.Context(), h.repo, id, req.Message, req.Kind)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toChatLogResponse(entry))
}

// List/Count are staff-only (see routes.go) — this is the "data để làm
// việc khác" read side: an admin/analytics view over every message
// shoppers have typed into the chat finder, filterable by kind (which
// internal path handled it) and a free-text search over the message body.
func (h *ChatLogHandler) List(w http.ResponseWriter, r *http.Request) {
	entries, err := application.GetChatLogs(r.Context(), h.repo, chatLogFilterFromQuery(r))
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toChatLogResponseList(entries))
}

func (h *ChatLogHandler) Count(w http.ResponseWriter, r *http.Request) {
	count, err := application.CountChatLogs(r.Context(), h.repo, chatLogFilterFromQuery(r))
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, map[string]int{"count": count})
}

func chatLogFilterFromQuery(r *http.Request) domain.ChatLogFilter {
	filter := domain.ChatLogFilter{
		Kind:   r.URL.Query().Get("kind"),
		Search: r.URL.Query().Get("search"),
	}
	if limit, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
		filter.Limit = limit
	}
	if offset, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil {
		filter.Offset = offset
	}
	return filter
}
