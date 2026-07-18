package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/product-qa/domain"
)

type questionResponse struct {
	ID           string     `json:"id"`
	ProductID    string     `json:"product_id"`
	AskerName    string     `json:"asker_name"`
	AskerEmail   *string    `json:"asker_email"`
	QuestionText string     `json:"question_text"`
	AnswerText   *string    `json:"answer_text"`
	Status       string     `json:"status"`
	IsPublished  bool       `json:"is_published"`
	AnsweredAt   *time.Time `json:"answered_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func toQuestionResponse(q *domain.Question) questionResponse {
	return questionResponse{
		ID: q.ID(), ProductID: q.ProductID(), AskerName: q.AskerName(), AskerEmail: q.AskerEmail(),
		QuestionText: q.QuestionText(), AnswerText: q.AnswerText(),
		Status: string(q.Status()), IsPublished: q.IsPublished(), AnsweredAt: q.AnsweredAt(),
		CreatedAt: q.CreatedAt(), UpdatedAt: q.UpdatedAt(),
	}
}

func toQuestionResponseList(questions []*domain.Question) []questionResponse {
	result := make([]questionResponse, len(questions))
	for i, q := range questions {
		result[i] = toQuestionResponse(q)
	}
	return result
}

type askQuestionRequest struct {
	AskerName    string  `json:"asker_name"`
	AskerEmail   *string `json:"asker_email"`
	QuestionText string  `json:"question_text"`
	// Website is a honeypot field — a real visitor never fills it in (hidden
	// via CSS on the FE), same anti-spam trick as internal/inquiry's Create.
	Website string `json:"website"`
}

type answerQuestionRequest struct {
	AnswerText string `json:"answer_text"`
}

type countResponse struct {
	Count int `json:"count"`
}
