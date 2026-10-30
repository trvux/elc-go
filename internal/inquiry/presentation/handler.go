package presentation

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	adsconversiondomain "github.com/trvux/elc-go/internal/adsconversion/domain"
	"github.com/trvux/elc-go/internal/inquiry/application"
	"github.com/trvux/elc-go/internal/inquiry/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/ratelimit"
	uploaddomain "github.com/trvux/elc-go/internal/upload/domain"
)

// InquiryHandler is the composition root for the inquiry module.
type InquiryHandler struct {
	repo          domain.InquiryRepository
	uploader      uploaddomain.Uploader
	notifier      adsconversiondomain.Notifier
	log           *zap.Logger
	createLimiter *ratelimit.Limiter
	uploadLimiter *ratelimit.Limiter
	clickLimiter  *ratelimit.Limiter
}

func NewInquiryHandler(repo domain.InquiryRepository, uploader uploaddomain.Uploader, notifier adsconversiondomain.Notifier, log *zap.Logger) *InquiryHandler {
	return &InquiryHandler{
		repo:     repo,
		uploader: uploader,
		notifier: notifier,
		log:      log,
		// 5 submissions per IP per 10 minutes — generous enough for a real
		// visitor to retry a typo, tight enough to blunt a naive spam script.
		createLimiter: ratelimit.New(5, 10*time.Minute),
		// More generous than createLimiter — a single visitor attaching 3-5
		// photos to one lead is normal use, not abuse.
		uploadLimiter: ratelimit.New(20, 10*time.Minute),
		// Most generous of the three — a visitor legitimately clicking
		// Zalo/Hotline on several product pages while browsing (or
		// double-clicking) is normal use, and RecordContactClick's own
		// session+channel dedup already collapses those into one lead.
		clickLimiter: ratelimit.New(30, 10*time.Minute),
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
		Name:        req.Name,
		Phone:       req.Phone,
		Email:       req.Email,
		Message:     req.Message,
		ProductID:   req.ProductID,
		ProjectID:   req.ProjectID,
		ServiceID:   req.ServiceID,
		LeadType:    domain.LeadType(req.LeadType),
		SubType:     req.SubType,
		QualifyData: req.QualifyData,
		Attachments: req.Attachments,
		Channel:     domain.ContactChannel(req.Channel),
		GCLID:       req.GCLID,
		UTMSource:   req.UTMSource,
		UTMMedium:   req.UTMMedium,
		UTMCampaign: req.UTMCampaign,
		UTMTerm:     req.UTMTerm,
		UTMContent:  req.UTMContent,
		GAClientID:  req.GAClientID,
		SourceIP:    &sourceIP,
		UserAgent:   &userAgent,
	}

	inquiry, err := application.CreateInquiry(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toInquiryResponse(inquiry))
}

// CreateClick handles a Zalo/Messenger/Hotline contact-link click — public,
// unauthenticated, same posture as Create but with no honeypot (see
// createClickRequest's doc comment) and a more generous rate limit.
// Expected to be called via navigator.sendBeacon from the client (fires
// even if the page is about to navigate away to zalo.me/m.me/tel:), so the
// response body is never actually read by the caller — still returns the
// full inquiryResponse for consistency/debuggability.
func (h *InquiryHandler) CreateClick(w http.ResponseWriter, r *http.Request) {
	var req createClickRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if !h.clickLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}

	sourceIP := httpserver.ClientIP(r)
	userAgent := r.UserAgent()

	input := domain.CreateContactClickInput{
		Channel:     domain.ContactChannel(req.Channel),
		ProductID:   req.ProductID,
		ProjectID:   req.ProjectID,
		ServiceID:   req.ServiceID,
		LeadType:    domain.LeadType(req.LeadType),
		SubType:     req.SubType,
		QualifyData: buildClickQualifyData(req.PagePath, req.EntityName),
		SessionID:   req.SessionID,
		GCLID:       req.GCLID,
		UTMSource:   req.UTMSource,
		UTMMedium:   req.UTMMedium,
		UTMCampaign: req.UTMCampaign,
		UTMTerm:     req.UTMTerm,
		UTMContent:  req.UTMContent,
		GAClientID:  req.GAClientID,
		SourceIP:    &sourceIP,
		UserAgent:   &userAgent,
	}

	inquiry, err := application.RecordContactClick(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toInquiryResponse(inquiry))
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

	inquiry, err := application.UpdateInquiryStatus(r.Context(), h.repo, h.notifier, h.log, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toInquiryResponse(inquiry))
}

// UpdateDetails handles PATCH /inquiries/{id} — staff filling in a click-
// origin lead's name/phone, and/or recording the order value. Separate
// from UpdateStatus (PATCH /inquiries/{id}/status) — see
// domain.UpdateInquiryDetailsInput's doc comment.
func (h *InquiryHandler) UpdateDetails(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateInquiryDetailsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := application.UpdateInquiryDetailsInput{
		ID:              id,
		Name:            req.Name,
		Phone:           req.Phone,
		ConversionValue: req.ConversionValue,
	}

	inquiry, err := application.UpdateInquiryDetails(r.Context(), h.repo, input)
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
