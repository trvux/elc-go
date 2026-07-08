package presentation

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/ratelimit"
	"github.com/trvux/elc-go/internal/review/application"
	"github.com/trvux/elc-go/internal/review/domain"
)

// ReviewHandler is the composition root for the review module.
type ReviewHandler struct {
	repo          domain.ReviewRepository
	createLimiter *ratelimit.Limiter
}

func NewReviewHandler(repo domain.ReviewRepository) *ReviewHandler {
	return &ReviewHandler{
		repo: repo,
		// 3 reviews per IP (or phone) per product/service per day — enough
		// for a genuine retry after a typo, tight enough to blunt a review-
		// bombing script. Keyed by "identifier:entityID", see Create.
		createLimiter: ratelimit.New(3, 24*time.Hour),
	}
}

// Create handles the public review submission. Not behind RequireAuth (any
// site visitor submits this) — the honeypot + rate limiter + blocklist
// auto-filter (domain.NewReview) are the actual defense.
func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if req.Website != "" {
		// Honeypot tripped — pretend success so the bot doesn't learn to
		// look for a different signal, but silently drop the submission.
		httpserver.WriteJSON(w, http.StatusCreated, publicReviewResponse{})
		return
	}

	entityID := ""
	if req.ProductID != nil {
		entityID = *req.ProductID
	} else if req.ServiceID != nil {
		entityID = *req.ServiceID
	}

	sourceIP := httpserver.ClientIP(r)
	if !h.createLimiter.Allow(sourceIP + ":" + entityID) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}
	if req.ReviewerPhone != nil && *req.ReviewerPhone != "" {
		if !h.createLimiter.Allow(*req.ReviewerPhone + ":" + entityID) {
			httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
			return
		}
	}

	userAgent := r.UserAgent()

	input := domain.CreateReviewInput{
		ProductID:     req.ProductID,
		ServiceID:     req.ServiceID,
		Rating:        req.Rating,
		Comment:       req.Comment,
		ReviewerName:  req.ReviewerName,
		ReviewerPhone: req.ReviewerPhone,
		SourceIP:      &sourceIP,
		UserAgent:     &userAgent,
	}

	review, err := application.CreateReview(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toPublicReviewResponse(review))
}

// ListPublic returns published reviews for a product or service — the
// public detail page's review list. Unpublished reviews (auto-hidden or
// manually hidden) never appear here.
func (h *ReviewHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	filter := parseReviewFilter(r)
	isPublished := true
	filter.IsPublished = &isPublished

	reviews, err := application.GetReviews(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toPublicReviewResponseList(reviews))
}

// Summary returns the aggregate rating for a product or service — used for
// star-rating display and schema.org AggregateRating JSON-LD.
func (h *ReviewHandler) Summary(w http.ResponseWriter, r *http.Request) {
	var productID, serviceID *string
	if v := r.URL.Query().Get("product_id"); v != "" {
		productID = &v
	}
	if v := r.URL.Query().Get("service_id"); v != "" {
		serviceID = &v
	}

	summary, err := application.GetReviewSummary(r.Context(), h.repo, productID, serviceID)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toReviewSummaryResponse(summary))
}

// List is the admin moderation screen's list — every review regardless of
// IsPublished, unless the caller explicitly filters.
func (h *ReviewHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := parseReviewFilter(r)
	if v := r.URL.Query().Get("is_published"); v != "" {
		isPublished := v == "true"
		filter.IsPublished = &isPublished
	}

	reviews, err := application.GetReviews(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toReviewResponseList(reviews))
}

func (h *ReviewHandler) Count(w http.ResponseWriter, r *http.Request) {
	filter := parseReviewFilter(r)
	if v := r.URL.Query().Get("is_published"); v != "" {
		isPublished := v == "true"
		filter.IsPublished = &isPublished
	}

	count, err := application.CountReviews(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, countResponse{Count: count})
}

func (h *ReviewHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	review, err := application.GetReviewByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if review == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("review"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toReviewResponse(review))
}

// UpdatePublished is the admin moderation screen's show/hide action.
func (h *ReviewHandler) UpdatePublished(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateReviewPublishedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	review, err := application.SetReviewPublished(r.Context(), h.repo, id, req.IsPublished)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toReviewResponse(review))
}

func (h *ReviewHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteReview(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseReviewFilter(r *http.Request) domain.ReviewFilter {
	filter := domain.ReviewFilter{}
	if v := r.URL.Query().Get("product_id"); v != "" {
		filter.ProductID = &v
	}
	if v := r.URL.Query().Get("service_id"); v != "" {
		filter.ServiceID = &v
	}
	return filter
}
