package presentation

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/faq/application"
	"github.com/trvux/elc-go/internal/faq/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

type FAQHandler struct {
	repo domain.FAQRepository
}

func NewFAQHandler(repo domain.FAQRepository) *FAQHandler {
	return &FAQHandler{repo: repo}
}

// List is the public read for one owner — published FAQs only, ordered for
// display. Feeds both the visible accordion and the FAQPage JSON-LD block
// on the same page (see shared/lib/seo-schema.ts's getFAQPage on the
// frontend) — same data, two renderings.
func (h *FAQHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.FAQFilter{
		OwnerType:     domain.OwnerType(chi.URLParam(r, "ownerType")),
		OwnerID:       chi.URLParam(r, "ownerID"),
		PublishedOnly: true,
	}
	if !filter.OwnerType.IsValid() {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{
			"ownerType": {"invalid owner type"},
		}))
		return
	}

	faqs, err := application.GetFAQsByOwner(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toFAQResponseList(faqs))
}

// AdminListByOwner is the staff-only read for one owner's FAQ management
// screen — everything, including unpublished drafts.
func (h *FAQHandler) AdminListByOwner(w http.ResponseWriter, r *http.Request) {
	filter := domain.FAQFilter{
		OwnerType:     domain.OwnerType(chi.URLParam(r, "ownerType")),
		OwnerID:       chi.URLParam(r, "ownerID"),
		PublishedOnly: false,
	}
	if !filter.OwnerType.IsValid() {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{
			"ownerType": {"invalid owner type"},
		}))
		return
	}

	faqs, err := application.GetFAQsByOwner(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toFAQResponseList(faqs))
}

func (h *FAQHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createFAQRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateFAQInput{
		OwnerType:   domain.OwnerType(req.OwnerType),
		OwnerID:     req.OwnerID,
		Question:    req.Question,
		Answer:      req.Answer,
		OrderIndex:  req.OrderIndex,
		IsPublished: req.IsPublished,
	}

	faq, err := application.CreateFAQ(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toFAQResponse(faq))
}

func (h *FAQHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateFAQRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateFAQInput{
		ID:          id,
		Question:    req.Question,
		Answer:      req.Answer,
		OrderIndex:  req.OrderIndex,
		IsPublished: req.IsPublished,
	}

	faq, err := application.UpdateFAQ(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toFAQResponse(faq))
}

func (h *FAQHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteFAQ(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
