package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// QuestionStatus tracks a guest-submitted product question through staff
// moderation. There is no "reopen" transition (unlike inquiry.InquiryStatus)
// — answering or rejecting is meant to be the end of the pipeline for a
// given question; staff can always answer again by editing AnswerText via a
// fresh Answer() call while still in pending, but once answered/rejected a
// question is done.
type QuestionStatus string

const (
	QuestionStatusPending  QuestionStatus = "pending"
	QuestionStatusAnswered QuestionStatus = "answered"
	QuestionStatusRejected QuestionStatus = "rejected"
)

func (s QuestionStatus) IsValid() bool {
	switch s {
	case QuestionStatusPending, QuestionStatusAnswered, QuestionStatusRejected:
		return true
	default:
		return false
	}
}

// Question is a guest-submitted question about a specific product, answered
// by staff. Unlike internal/inquiry (a private CRM lead pipeline), a
// Question's AnswerText is public-facing once IsPublished — this is the
// concept inquiry has no precedent for. Answering auto-publishes (no
// separate approval gate like product's draft/proposed/published workflow):
// only staff (content:write) can answer in the first place, so the answer
// has already been reviewed by the time it exists, unlike a brand-new
// product page which any employee could otherwise publish unreviewed.
type Question struct {
	id           string
	productID    string
	askerName    string
	askerEmail   *string
	questionText string
	answerText   *string
	status       QuestionStatus
	isPublished  bool
	answeredAt   *time.Time
	sourceIP     *string
	userAgent    *string
	createdAt    time.Time
	updatedAt    time.Time
	deletedAt    *time.Time
}

// NewQuestion validates and creates a new Question from a public form
// submission. sourceIP/userAgent are request metadata, not user input, so
// they're not validated — only recorded for later spam/abuse review, same as
// internal/inquiry.NewInquiry.
func NewQuestion(productID, askerName string, askerEmail *string, questionText string, sourceIP, userAgent *string) (*Question, error) {
	fields := map[string][]string{}

	if productID == "" {
		fields["product_id"] = []string{"product_id is required"}
	}
	if errs := validateAskerName(askerName); len(errs) > 0 {
		fields["asker_name"] = errs
	}
	if errs := validateAskerEmail(askerEmail); len(errs) > 0 {
		fields["asker_email"] = errs
	}
	if errs := validateQuestionText(questionText); len(errs) > 0 {
		fields["question_text"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &Question{
		productID:    productID,
		askerName:    askerName,
		askerEmail:   askerEmail,
		questionText: questionText,
		status:       QuestionStatusPending,
		isPublished:  false,
		sourceIP:     sourceIP,
		userAgent:    userAgent,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// RehydrateQuestion reconstructs from a trusted DB row — no validation. Only
// the infrastructure layer should call this.
func RehydrateQuestion(
	id, productID, askerName string,
	askerEmail *string,
	questionText string,
	answerText *string,
	status QuestionStatus,
	isPublished bool,
	answeredAt *time.Time,
	sourceIP, userAgent *string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *Question {
	return &Question{
		id: id, productID: productID, askerName: askerName, askerEmail: askerEmail,
		questionText: questionText, answerText: answerText,
		status: status, isPublished: isPublished, answeredAt: answeredAt,
		sourceIP: sourceIP, userAgent: userAgent,
		createdAt: createdAt, updatedAt: updatedAt, deletedAt: deletedAt,
	}
}

func (q *Question) ID() string             { return q.id }
func (q *Question) ProductID() string      { return q.productID }
func (q *Question) AskerName() string      { return q.askerName }
func (q *Question) AskerEmail() *string    { return q.askerEmail }
func (q *Question) QuestionText() string   { return q.questionText }
func (q *Question) AnswerText() *string    { return q.answerText }
func (q *Question) Status() QuestionStatus { return q.status }
func (q *Question) IsPublished() bool      { return q.isPublished }
func (q *Question) AnsweredAt() *time.Time { return q.answeredAt }
func (q *Question) SourceIP() *string      { return q.sourceIP }
func (q *Question) UserAgent() *string     { return q.userAgent }
func (q *Question) CreatedAt() time.Time   { return q.createdAt }
func (q *Question) UpdatedAt() time.Time   { return q.updatedAt }
func (q *Question) DeletedAt() *time.Time  { return q.deletedAt }

// Answer publishes staff's reply immediately — see the type doc comment for
// why this doesn't go through a separate approval gate.
func (q *Question) Answer(text string) error {
	if errs := validateAnswerText(text); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"answer_text": errs})
	}
	if q.status != QuestionStatusPending {
		return apperr.NewValidationError("validation failed", map[string][]string{
			"status": {"only a pending question can be answered"},
		})
	}
	now := time.Now()
	q.answerText = &text
	q.status = QuestionStatusAnswered
	q.isPublished = true
	q.answeredAt = &now
	q.updatedAt = now
	return nil
}

// Reject hides a question (spam/inappropriate) without answering it.
func (q *Question) Reject() error {
	if q.status != QuestionStatusPending {
		return apperr.NewValidationError("validation failed", map[string][]string{
			"status": {"only a pending question can be rejected"},
		})
	}
	q.status = QuestionStatusRejected
	q.isPublished = false
	q.updatedAt = time.Now()
	return nil
}

func (q *Question) MarkDeleted(deletedAt time.Time) {
	q.deletedAt = &deletedAt
}

func validateAskerName(name string) []string {
	var errs []string
	if strings.TrimSpace(name) == "" {
		errs = append(errs, "asker_name is required")
	} else if utf8.RuneCountInString(name) > 255 {
		errs = append(errs, "asker_name must not exceed 255 characters")
	}
	return errs
}

func validateAskerEmail(email *string) []string {
	if email == nil || *email == "" {
		return nil
	}
	if !strings.Contains(*email, "@") || !strings.Contains(*email, ".") {
		return []string{"asker_email is not valid"}
	}
	return nil
}

func validateQuestionText(text string) []string {
	var errs []string
	if strings.TrimSpace(text) == "" {
		errs = append(errs, "question_text is required")
	} else if utf8.RuneCountInString(text) > 2000 {
		errs = append(errs, "question_text must not exceed 2000 characters")
	}
	return errs
}

func validateAnswerText(text string) []string {
	var errs []string
	if strings.TrimSpace(text) == "" {
		errs = append(errs, "answer_text is required")
	} else if utf8.RuneCountInString(text) > 2000 {
		errs = append(errs, "answer_text must not exceed 2000 characters")
	}
	return errs
}

// CreateQuestionInput is the public-facing create payload.
type CreateQuestionInput struct {
	ProductID    string
	AskerName    string
	AskerEmail   *string
	QuestionText string
	SourceIP     *string
	UserAgent    *string
}

// QuestionFilter drives both the public per-product list (ProductID +
// PublishedOnly true) and the staff moderation queue (Status, no
// PublishedOnly restriction).
type QuestionFilter struct {
	ProductID     *string
	Status        *QuestionStatus
	PublishedOnly bool
	Limit         int
	Offset        int
}
