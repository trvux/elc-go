package presentation

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/trvux/elc-go/internal/event/application"
	"github.com/trvux/elc-go/internal/event/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/ratelimit"
)

type EventHandler struct {
	repo          domain.EventRepository
	createLimiter *ratelimit.Limiter
}

func NewEventHandler(repo domain.EventRepository) *EventHandler {
	return &EventHandler{
		repo: repo,
		// Looser than inquiry's create limiter — legit browsing across
		// several product pages in a short window is normal, not abuse.
		createLimiter: ratelimit.New(30, time.Minute),
	}
}

// Create is the public tracking-beacon endpoint — fire-and-forget from the
// caller's perspective (elc-tem's logEventAction ignores the response body),
// so this only ever returns a bare status code.
func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !h.createLimiter.Allow(httpserver.ClientIP(r)) {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	var req createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	var entityType *domain.EntityType
	if req.EntityType != nil {
		et := domain.EntityType(*req.EntityType)
		entityType = &et
	}

	err := application.LogEvent(r.Context(), h.repo, domain.CreateEventInput{
		Name:       domain.EventName(req.Name),
		EntityType: entityType,
		EntityID:   req.EntityID,
		PagePath:   req.PagePath,
		SessionID:  req.SessionID,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *EventHandler) TopViewed(w http.ResponseWriter, r *http.Request) {
	entityType := domain.EntityType(r.URL.Query().Get("entity_type"))
	if !entityType.IsValid() {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{
			"entity_type": {"required, one of product/project/service"},
		}))
		return
	}

	rows, err := application.GetTopViewed(r.Context(), h.repo, domain.TopViewedFilter{
		EntityType: entityType,
		Since:      time.Now().AddDate(0, 0, -30), // last 30 days — matches the dashboard's own recency window
		Limit:      10,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toTopViewedResponse(rows))
}
