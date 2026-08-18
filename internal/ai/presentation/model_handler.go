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

// ModelHandler is the admin CRUD surface for provider models — role
// ('chat'/'classifier'), pricing, fallback priority, default/active flags.
// Permission-gated in routes.go.
type ModelHandler struct {
	repo domain.ModelRepository
}

func NewModelHandler(repo domain.ModelRepository) *ModelHandler {
	return &ModelHandler{repo: repo}
}

func (h *ModelHandler) List(w http.ResponseWriter, r *http.Request) {
	models, err := application.ListModels(r.Context(), h.repo)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toModelResponseList(models))
}

func (h *ModelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	model, err := application.CreateModel(r.Context(), h.repo, application.CreateModelInput{
		ProviderID: req.ProviderID, ModelName: req.ModelName, DisplayName: req.DisplayName,
		Role: domain.ModelRole(req.Role), Pricing: req.Pricing, FallbackPriority: req.FallbackPriority,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusCreated, toModelResponse(model))
}

func (h *ModelHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	model, err := application.UpdateModel(r.Context(), h.repo, application.UpdateModelInput{
		ID: id, DisplayName: req.DisplayName, Pricing: req.Pricing,
		FallbackPriority: req.FallbackPriority, IsDefault: req.IsDefault, IsActive: req.IsActive,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toModelResponse(model))
}

func (h *ModelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := application.DeleteModel(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
