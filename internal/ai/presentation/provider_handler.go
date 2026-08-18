package presentation

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/ai/application"
	"github.com/trvux/elc-go/internal/ai/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

// ProviderHandler is the admin CRUD surface for configured LLM vendors —
// permission-gated in routes.go, never mounted for anonymous callers.
type ProviderHandler struct {
	repo domain.ProviderRepository
}

func NewProviderHandler(repo domain.ProviderRepository) *ProviderHandler {
	return &ProviderHandler{repo: repo}
}

func (h *ProviderHandler) List(w http.ResponseWriter, r *http.Request) {
	providers, err := application.ListProviders(r.Context(), h.repo)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toProviderResponseList(providers))
}

func (h *ProviderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	provider, err := application.CreateProvider(r.Context(), h.repo, application.CreateProviderInput{
		Name: req.Name, DisplayName: req.DisplayName, BaseURL: req.BaseURL,
		APIKey: req.APIKey, PricingDocURL: req.PricingDocURL,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, toProviderResponse(provider))
}

func (h *ProviderHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	provider, err := application.UpdateProvider(r.Context(), h.repo, application.UpdateProviderInput{
		ID: id, DisplayName: req.DisplayName, BaseURL: req.BaseURL,
		APIKey: req.APIKey, PricingDocURL: req.PricingDocURL, IsActive: req.IsActive,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toProviderResponse(provider))
}

func (h *ProviderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := application.DeleteProvider(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
