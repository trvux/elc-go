package presentation

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/ai/application"
	"github.com/trvux/elc-go/internal/ai/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// defaultUsageReportWindow is how far back GetUsageReport looks when the
// caller doesn't pass ?from — 30 days is enough for a monthly ROI check
// without the query scanning the whole table by default.
const defaultUsageReportWindow = 30 * 24 * time.Hour

// ReportHandler is the admin read-only surface (Phase 3) over what Phase 1/2
// already persist: cost/usage aggregates and conversation history.
// Permission-gated in routes.go, same as ProviderHandler/ModelHandler.
type ReportHandler struct {
	repo domain.ConversationRepository
}

func NewReportHandler(repo domain.ConversationRepository) *ReportHandler {
	return &ReportHandler{repo: repo}
}

func (h *ReportHandler) Usage(w http.ResponseWriter, r *http.Request) {
	toPtr, err := parseTimeParam(r.URL.Query().Get("to"))
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	fromPtr, err := parseTimeParam(r.URL.Query().Get("from"))
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	to := time.Now()
	if toPtr != nil {
		to = *toPtr
	}
	from := to.Add(-defaultUsageReportWindow)
	if fromPtr != nil {
		from = *fromPtr
	}

	groupBy := domain.UsageGroupBy(r.URL.Query().Get("groupBy"))
	if groupBy == "" {
		groupBy = domain.UsageGroupByDay
	}

	rows, err := application.GetUsageReport(r.Context(), h.repo, from, to, groupBy)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toUsageReportResponse(string(groupBy), from, to, rows))
}

func (h *ReportHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	from, err := parseTimeParam(r.URL.Query().Get("from"))
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	to, err := parseTimeParam(r.URL.Query().Get("to"))
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	limit, err := parseIntParam(r.URL.Query().Get("limit"), defaultConversationListLimit)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if limit <= 0 || limit > maxConversationListLimit {
		limit = defaultConversationListLimit
	}
	offset, err := parseIntParam(r.URL.Query().Get("offset"), 0)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if offset < 0 {
		offset = 0
	}

	summaries, total, err := application.ListConversations(r.Context(), h.repo, domain.ConversationFilter{
		From: from, To: to, Limit: limit, Offset: offset,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toConversationListResponse(summaries, total, limit, offset))
}

func (h *ReportHandler) ConversationMessages(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	messages, err := application.GetConversationMessages(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toConversationMessageDetailList(messages))
}

func parseTimeParam(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, apperr.NewValidationError("invalid date, expected RFC3339", nil)
	}
	return &t, nil
}

// parseIntParam mirrors parseTimeParam: empty means "use the caller's
// default", but a present, malformed value is a validation error, not a
// silent fallback — an admin who mistypes ?limit=abc should see that,
// not get a quietly-different page size.
func parseIntParam(v string, fallback int) (int, error) {
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, apperr.NewValidationError("invalid integer", nil)
	}
	return n, nil
}
