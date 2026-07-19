package presentation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/ratelimit"
	"github.com/trvux/elc-go/internal/wishlist/application"
	"github.com/trvux/elc-go/internal/wishlist/domain"
)

type WishlistHandler struct {
	repo         domain.WishlistRepository
	writeLimiter *ratelimit.Limiter
}

func NewWishlistHandler(repo domain.WishlistRepository) *WishlistHandler {
	return &WishlistHandler{
		repo: repo,
		// Generous — this defends against a scripted abuse loop, not real
		// visitor usage (same reasoning as inquiry's createLimiter).
		writeLimiter: ratelimit.New(60, 10*time.Minute),
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

func (h *WishlistHandler) List(w http.ResponseWriter, r *http.Request) {
	id, ok := visitorID(w, r)
	if !ok {
		return
	}

	items, err := application.ListWishlistItems(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toWishlistItemResponseList(items))
}

func (h *WishlistHandler) Add(w http.ResponseWriter, r *http.Request) {
	id, ok := visitorID(w, r)
	if !ok {
		return
	}

	if !h.writeLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}

	var req addWishlistItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	item, err := application.AddWishlistItem(r.Context(), h.repo, id, req.ProductID)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toWishlistItemResponse(&domain.WishlistItemWithProduct{WishlistItem: item}))
}

func (h *WishlistHandler) Remove(w http.ResponseWriter, r *http.Request) {
	id, ok := visitorID(w, r)
	if !ok {
		return
	}

	if !h.writeLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}

	productID := chi.URLParam(r, "productId")

	if err := application.RemoveWishlistItem(r.Context(), h.repo, id, productID); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
