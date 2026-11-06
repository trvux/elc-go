package presentation

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
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
	files         uploaddomain.FileStore
	notifier      adsconversiondomain.Notifier
	log           *zap.Logger
	createLimiter *ratelimit.Limiter
	uploadLimiter *ratelimit.Limiter
	fileLimiter   *ratelimit.Limiter
	fileSlots     chan struct{}
	clickLimiter  *ratelimit.Limiter
}

func NewInquiryHandler(repo domain.InquiryRepository, uploader uploaddomain.Uploader, files uploaddomain.FileStore, notifier adsconversiondomain.Notifier, log *zap.Logger) *InquiryHandler {
	return &InquiryHandler{
		repo:      repo,
		uploader:  uploader,
		files:     files,
		fileSlots: make(chan struct{}, maxConcurrentFileUploads),
		notifier:  notifier,
		log:       log,
		// 5 submissions per IP per 10 minutes — generous enough for a real
		// visitor to retry a typo, tight enough to blunt a naive spam script.
		createLimiter: ratelimit.New(5, 10*time.Minute),
		// More generous than createLimiter — a single visitor attaching 3-5
		// photos to one lead is normal use, not abuse.
		uploadLimiter: ratelimit.New(20, 10*time.Minute),
		// Stricter than uploadLimiter: files are up to 15 MB each, and six is the most one
		// lead can carry, so ten in ten minutes covers a retry without inviting a flood.
		fileLimiter: ratelimit.New(10, 10*time.Minute),
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

	verified, err := h.verifyAttachments(r, input.Attachments)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	input.Attachments = verified

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

// maxExportRows keeps one download bounded: 58 leads exist today, so this is
// ~170x headroom, and the cap makes a runaway request cheap to reason about.
const maxExportRows = 10000

// Export returns the full inquiry rows for the admin's Excel download. Same
// JSON as List (the FE builds the workbook, because the Vietnamese labels
// for qualify answers and the lead-priority badges live there), but admin
// only and capped. Each call is logged: customer contact data left the
// system and someone should be able to see who took it and when.
func (h *InquiryHandler) Export(w http.ResponseWriter, r *http.Request) {
	filter := parseInquiryFilter(r)
	filter.Limit = maxExportRows

	inquiries, err := application.GetInquiries(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	userID, _ := httpserver.UserIDFromContext(r.Context())
	h.log.Info("inquiries exported",
		zap.String("user_id", userID),
		zap.Int("rows", len(inquiries)),
		zap.String("status", filter.Status),
		zap.String("search", filter.Search),
		zap.Time("from", filter.From),
		zap.Time("to", filter.To),
		zap.Int("selected_ids", len(filter.IDs)),
	)

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
	q := r.URL.Query()
	filter := domain.InquiryFilter{
		Status: q.Get("status"),
		Search: q.Get("search"),
	}
	filter.From, filter.To = parseDateRange(q.Get("from"), q.Get("to"))
	filter.IDs = parseIDList(q.Get("ids"))
	return filter
}

// vnLocation is the business's calendar: "from=2026-11-01" means midnight in
// Vietnam, not UTC, otherwise leads from the first 7 hours of the day land in
// the previous day's export.
var vnLocation = time.FixedZone("ICT", 7*60*60)

// parseDateRange reads YYYY-MM-DD bounds; both ends are inclusive days, so To
// becomes the start of the following day. A malformed value is ignored (no
// bound) rather than failing the whole list.
func parseDateRange(from, to string) (time.Time, time.Time) {
	var start, end time.Time
	if t, err := time.ParseInLocation("2006-01-02", from, vnLocation); err == nil {
		start = t
	}
	if t, err := time.ParseInLocation("2006-01-02", to, vnLocation); err == nil {
		end = t.AddDate(0, 0, 1)
	}
	return start, end
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// maxExportIDs bounds the query string; the FE sends selected rows only.
const maxExportIDs = 500

// parseIDList keeps only well-formed UUIDs (a bad one would otherwise make
// the ::uuid[] cast fail the whole query).
func parseIDList(raw string) []string {
	if raw == "" {
		return nil
	}
	var ids []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if uuidPattern.MatchString(part) {
			ids = append(ids, part)
		}
		if len(ids) == maxExportIDs {
			break
		}
	}
	return ids
}
