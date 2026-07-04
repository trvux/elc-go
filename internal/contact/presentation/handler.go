package presentation

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/contact/application"
	"github.com/trvux/elc-go/internal/contact/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// ContactHandler is the composition root for the contact module: it is the
// only place allowed to hold a concrete domain.ContactRepository and wire it
// into each application use case.
type ContactHandler struct {
	repo domain.ContactRepository
}

func NewContactHandler(repo domain.ContactRepository) *ContactHandler {
	return &ContactHandler{repo: repo}
}

func (h *ContactHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.ContactFilter{
		Type:   r.URL.Query().Get("type"),
		Search: r.URL.Query().Get("search"),
	}

	contacts, err := application.GetContacts(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toContactResponseList(contacts))
}

func (h *ContactHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	contact, err := application.GetContactByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if contact == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("contact"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toContactResponse(contact))
}

func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateContactInput{
		Type:       req.Type,
		Label:      req.Label,
		Value:      req.Value,
		IsActive:   req.IsActive,
		OrderIndex: req.OrderIndex,
	}

	contact, err := application.CreateContact(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toContactResponse(contact))
}

func (h *ContactHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateContactInput{
		ID:         id,
		Type:       req.Type,
		Label:      req.Label,
		Value:      req.Value,
		IsActive:   req.IsActive,
		OrderIndex: req.OrderIndex,
	}

	contact, err := application.UpdateContact(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toContactResponse(contact))
}

func (h *ContactHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteContact(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
