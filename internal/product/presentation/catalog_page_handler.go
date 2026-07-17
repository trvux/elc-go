package presentation

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/product/application"
	"github.com/trvux/elc-go/internal/product/domain"
)

type catalogPageResponse struct {
	Content         json.RawMessage `json:"content"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func toCatalogPageResponse(p *domain.CatalogPage) catalogPageResponse {
	return catalogPageResponse{
		Content: p.Content(), MetaTitle: p.MetaTitle(), MetaDescription: p.MetaDescription(),
		UpdatedAt: p.UpdatedAt(),
	}
}

type updateCatalogPageRequest struct {
	Content         json.RawMessage `json:"content"`
	MetaTitle       *string         `json:"meta_title"`
	MetaDescription *string         `json:"meta_description"`
}

type CatalogPageHandler struct {
	repo domain.CatalogPageRepository
}

func NewCatalogPageHandler(repo domain.CatalogPageRepository) *CatalogPageHandler {
	return &CatalogPageHandler{repo: repo}
}

func (h *CatalogPageHandler) Get(w http.ResponseWriter, r *http.Request) {
	p, err := application.GetCatalogPage(r.Context(), h.repo)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toCatalogPageResponse(p))
}

func (h *CatalogPageHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateCatalogPageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	p, err := application.UpdateCatalogPage(r.Context(), h.repo, domain.UpdateCatalogPageInput{
		Content: req.Content, MetaTitle: req.MetaTitle, MetaDescription: req.MetaDescription,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.WriteJSON(w, http.StatusOK, toCatalogPageResponse(p))
}

func RegisterCatalogPageRoutes(r chi.Router, h *CatalogPageHandler, verifier httpserver.TokenVerifier) {
	r.Route("/catalog-page", func(r chi.Router) {
		r.Get("/", h.Get)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireAuth(verifier))
			r.Use(httpserver.RequirePermission(authdomain.CanWriteContent))
			r.Put("/", h.Update)
		})
	})
}
