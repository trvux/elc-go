package presentation

import (
	"encoding/json"
	"net/http"

	"github.com/trvux/elc-go/internal/contentvar/application"
	"github.com/trvux/elc-go/internal/contentvar/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
)

type ContentVarHandler struct {
	resolver *application.Resolver
}

func NewContentVarHandler(resolver *application.Resolver) *ContentVarHandler {
	return &ContentVarHandler{resolver: resolver}
}

type resolveVariablesRequest struct {
	Variables []variableRequestDTO `json:"variables"`
}

type variableRequestDTO struct {
	ID     string            `json:"id"`
	Metric string            `json:"metric"`
	Filter variableFilterDTO `json:"filter"`
}

type variableFilterDTO struct {
	GroupSlug      string `json:"groupSlug"`
	CategorySlug   string `json:"categorySlug"`
	BrandSlug      string `json:"brandSlug"`
	AttributeCode  string `json:"attributeCode"`
	AttributeValue string `json:"attributeValue"`
}

type resolveVariablesResponse struct {
	Values map[string]any `json:"values"`
}

// ResolveVariables batch-resolves Tiptap content-variable placeholders
// (product count / price range for a category, group or brand scope)
// against the live catalog in one round trip — called by the frontend's
// SSR render step right before serving a /san-pham page, so the numbers
// baked into the response HTML always match the current catalog instead
// of whatever a content author typed at save time.
func (h *ContentVarHandler) ResolveVariables(w http.ResponseWriter, r *http.Request) {
	var req resolveVariablesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}
	if len(req.Variables) == 0 {
		httpserver.WriteJSON(w, http.StatusOK, resolveVariablesResponse{Values: map[string]any{}})
		return
	}

	requests := make([]domain.VariableRequest, len(req.Variables))
	for i, v := range req.Variables {
		if v.ID == "" {
			httpserver.WriteError(w, apperr.NewValidationError("invalid variable", map[string][]string{
				"variables": {"id is required"},
			}))
			return
		}
		metric := domain.Metric(v.Metric)
		if !metric.IsValid() {
			httpserver.WriteError(w, apperr.NewValidationError("invalid variable", map[string][]string{
				"variables": {"unknown metric: " + v.Metric},
			}))
			return
		}
		requests[i] = domain.VariableRequest{
			ID:     v.ID,
			Metric: metric,
			Filter: domain.VariableFilter{
				GroupSlug:      v.Filter.GroupSlug,
				CategorySlug:   v.Filter.CategorySlug,
				BrandSlug:      v.Filter.BrandSlug,
				AttributeCode:  v.Filter.AttributeCode,
				AttributeValue: v.Filter.AttributeValue,
			},
		}
	}

	values, err := h.resolver.Resolve(r.Context(), requests)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, resolveVariablesResponse{Values: values})
}
