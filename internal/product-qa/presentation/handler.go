package presentation

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/ratelimit"
	"github.com/trvux/elc-go/internal/product-qa/application"
	"github.com/trvux/elc-go/internal/product-qa/domain"
)

type QuestionHandler struct {
	repo       domain.QuestionRepository
	askLimiter *ratelimit.Limiter
}

func NewQuestionHandler(repo domain.QuestionRepository) *QuestionHandler {
	return &QuestionHandler{
		repo: repo,
		// Same limit/window as inquiry's createLimiter — generous enough for
		// a real visitor to retry, tight enough to blunt a naive spam script.
		askLimiter: ratelimit.New(5, 10*time.Minute),
	}
}

// Ask handles the public per-product question submission. Not behind
// RequireAuth (anonymous site visitors submit this) — the honeypot + rate
// limiter are the actual defense, same as internal/inquiry.Create.
func (h *QuestionHandler) Ask(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")

	var req askQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	if req.Website != "" {
		// Honeypot tripped — pretend success so the bot doesn't learn to look
		// for a different signal, but silently drop the submission.
		httpserver.WriteJSON(w, http.StatusCreated, questionResponse{})
		return
	}

	if !h.askLimiter.Allow(httpserver.ClientIP(r)) {
		httpserver.WriteError(w, apperr.NewTooManyRequestsError("please try again later"))
		return
	}

	sourceIP := httpserver.ClientIP(r)
	userAgent := r.UserAgent()

	input := domain.CreateQuestionInput{
		ProductID: productID, AskerName: req.AskerName, AskerEmail: req.AskerEmail,
		QuestionText: req.QuestionText, SourceIP: &sourceIP, UserAgent: &userAgent,
	}

	q, err := application.AskQuestion(r.Context(), h.repo, input)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusCreated, toQuestionResponse(q))
}

// ListForProduct is the public per-product read — only ever returns
// published answers, never pending/rejected ones.
func (h *QuestionHandler) ListForProduct(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productId")

	filter := domain.QuestionFilter{ProductID: &productID, PublishedOnly: true}
	questions, err := application.ListQuestions(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toQuestionResponseList(questions))
}

// List is the staff moderation queue — filterable by status (typically
// "pending"), across every product.
func (h *QuestionHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.QuestionFilter{}
	if v := r.URL.Query().Get("status"); v != "" {
		status := domain.QuestionStatus(v)
		if !status.IsValid() {
			httpserver.WriteError(w, apperr.NewValidationError("validation failed", map[string][]string{
				"status": {"invalid status value"},
			}))
			return
		}
		filter.Status = &status
	}
	if v := r.URL.Query().Get("product_id"); v != "" {
		filter.ProductID = &v
	}

	questions, err := application.ListQuestions(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toQuestionResponseList(questions))
}

func (h *QuestionHandler) Count(w http.ResponseWriter, r *http.Request) {
	filter := domain.QuestionFilter{}
	if v := r.URL.Query().Get("status"); v != "" {
		status := domain.QuestionStatus(v)
		filter.Status = &status
	}

	count, err := application.CountQuestions(r.Context(), h.repo, filter)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, countResponse{Count: count})
}

func (h *QuestionHandler) Answer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req answerQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, apperr.NewValidationError("invalid JSON body", nil))
		return
	}

	q, err := application.AnswerQuestion(r.Context(), h.repo, id, req.AnswerText)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toQuestionResponse(q))
}

func (h *QuestionHandler) Reject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	q, err := application.RejectQuestion(r.Context(), h.repo, id)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}

	httpserver.WriteJSON(w, http.StatusOK, toQuestionResponse(q))
}

func (h *QuestionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := application.DeleteQuestion(r.Context(), h.repo, id); err != nil {
		httpserver.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
