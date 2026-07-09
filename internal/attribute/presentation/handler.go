package presentation

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/attribute/application"
	"github.com/trvux/elc-go/internal/attribute/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

type AttributeDefinitionHandler struct {
	repo domain.AttributeDefinitionRepository
}

func NewAttributeDefinitionHandler(repo domain.AttributeDefinitionRepository) *AttributeDefinitionHandler {
	return &AttributeDefinitionHandler{repo: repo}
}

func (h *AttributeDefinitionHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.AttributeDefinitionFilter{
		IncludeGlobal:  r.URL.Query().Get("include_global") == "true",
		IncludeDeleted: r.URL.Query().Get("include_deleted") == "true",
	}
	if categoryID := r.URL.Query().Get("category_id"); categoryID != "" {
		filter.CategoryID = &categoryID
	}

	defs, err := application.GetAttributeDefinitions(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toAttributeDefinitionResponseList(defs))
}

func (h *AttributeDefinitionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	d, err := application.GetAttributeDefinitionByID(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	if d == nil {
		httpserver.WriteError(w, apperr.NewNotFoundError("attribute_definition"))
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toAttributeDefinitionResponse(d))
}

func (h *AttributeDefinitionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createAttributeDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.CreateAttributeDefinitionInput{
		CategoryID: req.CategoryID, Code: req.Code, Name: req.Name, GroupLabel: req.GroupLabel,
		DataType: req.DataType, Unit: req.Unit, Options: req.Options, OrderIndex: req.OrderIndex, IsRequired: req.IsRequired,
	}

	d, err := application.CreateAttributeDefinition(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toAttributeDefinitionResponse(d))
}

func (h *AttributeDefinitionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateAttributeDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	input := domain.UpdateAttributeDefinitionInput{
		ID: id, Name: req.Name, GroupLabel: req.GroupLabel, Unit: req.Unit,
		Options: req.Options, OrderIndex: req.OrderIndex, IsRequired: req.IsRequired,
	}

	d, err := application.UpdateAttributeDefinition(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toAttributeDefinitionResponse(d))
}

func (h *AttributeDefinitionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteAttributeDefinition(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AttributeDefinitionHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.RestoreAttributeDefinition(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
