package presentation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/ratelimit"
	"github.com/trvux/elc-go/internal/recently-viewed/application"
	"github.com/trvux/elc-go/internal/recently-viewed/domain"
)

type RecentlyViewedHandler struct {
	repo         domain.RecentlyViewedRepository
	writeLimiter *ratelimit.Limiter
}

func NewRecentlyViewedHandler(repo domain.RecentlyViewedRepository) *RecentlyViewedHandler {
	return &RecentlyViewedHandler{
		repo: repo,
		// Generous — defends against a scripted abuse loop, not real
		// visitor usage (same reasoning as inquiry's createLimiter).
		writeLimiter: ratelimit.New(120, 10*time.Minute),
	}
}

// visitorID reads the ID EnsureVisitorID (mounted in routes.go) attached to
// the request context. Missing means a routing mistake (the middleware
// wasn't mounted), not a client error, hence the 500 rather than a 400.
func visitorID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := httpserver.VisitorIDFromContext(r.Context())
	if !ok || id == "" {
		httpserver.WriteError(w, apperr.NewInternalError(fmt.Errorf("visitor_id missing from context")))
		return "", false
	}
	return id, true
}

func (h *RecentlyViewedHandler) List(w http.ResponseWriter, r *http.Request) {
	id, ok := visitorID(w, r)
	if !ok {
		return
	}

	items, err := application.ListRecentlyViewed(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toRecentlyViewedItemResponseList(items))
}

func (h *RecentlyViewedHandler) Record(w http.ResponseWriter, r *http.Request) {
	id, ok := visitorID(w, r)
	if !ok {
		return
	}

	if !h.writeLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}

	var req recordRecentlyViewedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	item, err := application.RecordRecentlyViewed(r.Context(), h.repo, id, req.ProductID)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toRecentlyViewedItemResponse(&domain.RecentlyViewedItemWithProduct{RecentlyViewedItem: item}))
}
