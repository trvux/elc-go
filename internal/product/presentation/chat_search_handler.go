package presentation

import (
	"encoding/json"
	"net/http"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/product/application"
)

type chatSearchRequest struct {
	Message string `json:"message"`
}

// chatSearchResponse extends the plain product-list shape with
// Explanation — a client-facing, template-assembled (not LLM-generated)
// Vietnamese sentence explaining why these particular filters were chosen
// (see application.ChatSearchResult/buildExplanation), e.g. "Gợi ý máy
// lạnh cho nhà hàng/quán ăn (âm trần), công suất đề xuất 2 HP (đã cộng
// thêm 40% do tải nhiệt cao hơn bình thường), thương hiệu Daikin, giá
// dưới 20 triệu." So a shopper sees the reasoning, not just a bare list.
type chatSearchResponse struct {
	productListResponse
	Explanation string `json:"explanation,omitempty"`
	// Suggestions are follow-up queries a shopper can tap to keep
	// narrowing (see application.buildSuggestions) — self-contained query
	// text, ready to send back as the next turn's message as-is.
	Suggestions []string `json:"suggestions,omitempty"`
}

// ChatSearch is the conversational counterpart to List: instead of
// query-param facets, it takes one free-text shopper message and returns a
// short curated list of matches (see application.ChatSearchProducts for
// how the message is parsed — no LLM/embeddings, a rule-based rewrite into
// the same ProductFilter the regular listing page uses). Public and
// read-only, same trust level as GetByIDsBatch.
func (h *ProductHandler) ChatSearch(w http.ResponseWriter, r *http.Request) {
	if !h.chatSearchLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}

	var req chatSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}
	if req.Message == "" {
		httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{
			"message": {"message is required"},
		}))
		return
	}

	result, err := application.ChatSearchProducts(r.Context(), h.repo, req.Message)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, chatSearchResponse{
		productListResponse: toProductListResponse(result.ProductListResult),
		Explanation:         result.Explanation,
		Suggestions:         result.Suggestions,
	})
}
