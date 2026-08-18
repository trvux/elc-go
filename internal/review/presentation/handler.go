package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/ratelimit"
	"github.com/trvux/elc-go/internal/review/application"
	"github.com/trvux/elc-go/internal/review/domain"
)

// entityColumnFor maps the {entityType} URL segment to both the reviews
// table column to query/write and which CreateReviewInput field to set.
// The only place request input picks a SQL identifier — validated against
// this fixed whitelist before it ever reaches a query (see
// infrastructure.PostgresReviewRepository.GetByEntity's doc comment).
func entityColumnFor(entityType string) (string, bool) {
	switch entityType {
	case "product":
		return "product_id", true
	case "project":
		return "project_id", true
	case "service":
		return "service_id", true
	case "news":
		return "news_id", true
	default:
		return "", false
	}
}

type ReviewHandler struct {
	repo          domain.ReviewRepository
	createLimiter *ratelimit.Limiter
}

func NewReviewHandler(repo domain.ReviewRepository) *ReviewHandler {
	return &ReviewHandler{
		repo: repo,
		// Same limit/window as inquiry/product-qa's create limiters —
		// generous enough for a real visitor to retry, tight enough to
		// blunt a naive spam script.
		createLimiter: ratelimit.New(5, 10*time.Minute),
	}
}

// Create handles the public review submission for one entity. Not behind
// RequireAuth (anonymous site visitors submit this) — the honeypot + rate
// limiter are the actual defense, same as internal/inquiry.Create.
func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entityType")
	entityColumn, ok := entityColumnFor(entityType)
	if !ok {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{
			"entityType": {"must be one of product, project, service, news"},
		}))
		return
	}
	entityID := chi.URLParam(r, "entityID")
	if _, err := uuid.Parse(entityID); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{
			"entityID": {"must be a valid UUID"},
		}))
		return
	}

	var req createReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if req.Website != "" {
		// Honeypot tripped — pretend success so the bot doesn't learn to
		// look for a different signal, but silently drop the submission.
		httpserver.WriteJSON(w, http.StatusCreated, reviewResponse{})
		return
	}

	if !h.createLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}

	sourceIP := httpserver.ClientIP(r)
	userAgent := r.UserAgent()

	input := domain.CreateReviewInput{
		Rating:        req.Rating,
		Comment:       req.Comment,
		ReviewerName:  req.ReviewerName,
		ReviewerPhone: req.ReviewerPhone,
		SourceIP:      &sourceIP,
		UserAgent:     &userAgent,
	}
	switch entityColumn {
	case "product_id":
		input.ProductID = &entityID
	case "project_id":
		input.ProjectID = &entityID
	case "service_id":
		input.ServiceID = &entityID
	case "news_id":
		input.NewsID = &entityID
	}

	review, err := application.CreateReview(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toReviewResponse(review))
}

// List is the public read for one entity — only ever published reviews,
// plus the aggregate rating computed over them.
func (h *ReviewHandler) List(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entityType")
	entityColumn, ok := entityColumnFor(entityType)
	if !ok {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{
			"entityType": {"must be one of product, project, service, news"},
		}))
		return
	}
	entityID := chi.URLParam(r, "entityID")
	if _, err := uuid.Parse(entityID); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{
			"entityID": {"must be a valid UUID"},
		}))
		return
	}

	reviews, aggregate, err := application.ListPublishedReviews(r.Context(), h.repo, entityColumn, entityID)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, reviewListResponse{
		Data:      toReviewResponseList(reviews),
		Aggregate: toReviewAggregateResponse(aggregate),
	})
}

func parseReviewFilter(r *http.Request) (domain.ReviewFilter, error) {
	filter := domain.ReviewFilter{}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			return filter, apperr.NewValidationError("invalid limit query param", nil)
		}
		filter.Limit = limit
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil {
			return filter, apperr.NewValidationError("invalid offset query param", nil)
		}
		filter.Offset = offset
	}
	return filter, nil
}

// AdminList is the staff-only list of every review — any entity type,
// published or not — newest first, with reviewer_phone and the reviewed
// product's name/slug included. Gated by RequireAuth+RequirePermission in
// routes.go; never mount this handler without that group.
func (h *ReviewHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	filter, err := parseReviewFilter(r)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	reviews, err := application.GetReviews(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toAdminReviewResponseList(reviews))
}

func (h *ReviewHandler) AdminCount(w http.ResponseWriter, r *http.Request) {
	filter, err := parseReviewFilter(r)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	count, err := application.CountReviews(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, reviewCountResponse{Count: count})
}
