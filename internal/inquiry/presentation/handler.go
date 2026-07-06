package presentation

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/inquiry/application"
	"github.com/trvux/elc-go/internal/inquiry/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/ratelimit"
)

// InquiryHandler is the composition root for the inquiry module.
type InquiryHandler struct {
	repo          domain.InquiryRepository
	notifier      domain.LeadNotifier
	createLimiter *ratelimit.Limiter
}

func NewInquiryHandler(repo domain.InquiryRepository, notifier domain.LeadNotifier) *InquiryHandler {
	return &InquiryHandler{
		repo:     repo,
		notifier: notifier,
		// 5 submissions per IP per 10 minutes — generous enough for a real
		// visitor to retry a typo, tight enough to blunt a naive spam script.
		createLimiter: ratelimit.New(5, 10*time.Minute),
	}
}

type countResponse struct {
	Count int `json:"count"`
}

// Create handles the public lead-capture submission. Not behind RequireAuth
// (anonymous site visitors submit this) — the honeypot + rate limiter are
// the actual defense, see internal/inquiry/domain's package doc.
func (h *InquiryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createInquiryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if req.Website != "" {
		// Honeypot tripped — pretend success so the bot doesn't learn to
		// look for a different signal, but silently drop the submission.
		httpserver.WriteJSON(w, http.StatusCreated, inquiryResponse{})
		return
	}

	if !h.createLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}

	sourceIP := httpserver.ClientIP(r)
	userAgent := r.UserAgent()

	input := domain.CreateInquiryInput{
		Name:      req.Name,
		Phone:     req.Phone,
		Email:     req.Email,
		Message:   req.Message,
		ProductID: req.ProductID,
		ProjectID: req.ProjectID,
		ServiceID: req.ServiceID,
		SourceIP:  &sourceIP,
		UserAgent: &userAgent,
	}

	inquiry, err := application.CreateInquiry(r.Context(), h.repo, h.notifier, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toInquiryResponse(inquiry))
}

func (h *InquiryHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := parseInquiryFilter(r)

	inquiries, err := application.GetInquiries(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toInquiryResponseList(inquiries))
}

func (h *InquiryHandler) Count(w http.ResponseWriter, r *http.Request) {
	filter := parseInquiryFilter(r)

	count, err := application.CountInquiries(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, countResponse{Count: count})
}

func (h *InquiryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	inquiry, err := application.GetInquiryByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if inquiry == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("inquiry"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toInquiryResponse(inquiry))
}

func (h *InquiryHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateInquiryStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := application.UpdateInquiryStatusInput{
		ID:           id,
		Status:       domain.InquiryStatus(req.Status),
		InternalNote: req.InternalNote,
	}

	inquiry, err := application.UpdateInquiryStatus(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toInquiryResponse(inquiry))
}

func parseInquiryFilter(r *http.Request) domain.InquiryFilter {
	return domain.InquiryFilter{
		Status: r.URL.Query().Get("status"),
		Search: r.URL.Query().Get("search"),
	}
}
