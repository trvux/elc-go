package domain

import (
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// OwnerType names which kind of page a FAQ belongs to — the polymorphic
// half of (owner_type, owner_id), same shape as internal/slug-registry's
// (entity_type, entity_id) rather than one nullable FK column per owner
// kind like internal/review — see docs/rfc (mã 8823 decision, "FAQ có tập
// owner lớn hơn reviews nhiều, mỗi loại thêm vào kiểu FK-cột-riêng là 1
// migration"). A fixed, closed set — not free text — enforced here AND by
// a DB CHECK constraint as a second line of defense.
type OwnerType string

const (
	OwnerTypeSystemPage OwnerType = "system_page"
	OwnerTypeProduct    OwnerType = "product"
	OwnerTypeProject    OwnerType = "project"
	OwnerTypeService    OwnerType = "service"
	OwnerTypeNews       OwnerType = "news"
	OwnerTypeBranch     OwnerType = "branch"
	OwnerTypePage       OwnerType = "page"
)

func (o OwnerType) IsValid() bool {
	switch o {
	case OwnerTypeSystemPage, OwnerTypeProduct, OwnerTypeProject, OwnerTypeService,
		OwnerTypeNews, OwnerTypeBranch, OwnerTypePage:
		return true
	default:
		return false
	}
}

// FAQ is one question/answer pair attached to exactly one owner page/entity
// — see mã 8823's decision doc for why this is a single polymorphic type
// shared across every owner kind instead of a per-module redefinition.
// Each entity/page writes its own FAQs independently — by design, questions
// are NOT shared/reused across owners even when the wording would overlap
// (e.g. "ELC bảo hành bao lâu" on both a product and a service) because the
// 4 content verticals (dự án B2B, sản phẩm B2C, dịch vụ B2C/B2B, blog) serve
// different intents/business models, so an answer tuned for one is rarely
// the right answer verbatim for another.
type FAQ struct {
	id          string
	ownerType   OwnerType
	ownerID     string
	question    string
	answer      string
	orderIndex  int
	isPublished bool
	createdAt   time.Time
	updatedAt   time.Time
}

// NewFAQ validates and creates a new FAQ from admin input.
func NewFAQ(ownerType OwnerType, ownerID, question, answer string, orderIndex int, isPublished bool) (*FAQ, error) {
	fields := map[string][]string{}

	if !ownerType.IsValid() {
		fields["ownerType"] = []string{"invalid owner type"}
	}
	if strings.TrimSpace(ownerID) == "" {
		fields["ownerId"] = []string{"ownerId is required"}
	}
	if errs := validateQuestion(question); len(errs) > 0 {
		fields["question"] = errs
	}
	if errs := validateAnswer(answer); len(errs) > 0 {
		fields["answer"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &FAQ{
		ownerType:   ownerType,
		ownerID:     ownerID,
		question:    question,
		answer:      answer,
		orderIndex:  orderIndex,
		isPublished: isPublished,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// RehydrateFAQ reconstructs from a trusted DB row — no validation. Only the
// infrastructure layer should call this.
func RehydrateFAQ(
	id string,
	ownerType OwnerType,
	ownerID, question, answer string,
	orderIndex int,
	isPublished bool,
	createdAt, updatedAt time.Time,
) *FAQ {
	return &FAQ{
		id:          id,
		ownerType:   ownerType,
		ownerID:     ownerID,
		question:    question,
		answer:      answer,
		orderIndex:  orderIndex,
		isPublished: isPublished,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

func (f *FAQ) ID() string           { return f.id }
func (f *FAQ) OwnerType() OwnerType { return f.ownerType }
func (f *FAQ) OwnerID() string      { return f.ownerID }
func (f *FAQ) Question() string     { return f.question }
func (f *FAQ) Answer() string       { return f.answer }
func (f *FAQ) OrderIndex() int      { return f.orderIndex }
func (f *FAQ) IsPublished() bool    { return f.isPublished }
func (f *FAQ) CreatedAt() time.Time { return f.createdAt }
func (f *FAQ) UpdatedAt() time.Time { return f.updatedAt }

func (f *FAQ) UpdateQuestion(question string) error {
	if errs := validateQuestion(question); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"question": errs})
	}
	f.question = question
	f.updatedAt = time.Now()
	return nil
}

func (f *FAQ) UpdateAnswer(answer string) error {
	if errs := validateAnswer(answer); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"answer": errs})
	}
	f.answer = answer
	f.updatedAt = time.Now()
	return nil
}

func (f *FAQ) Reorder(orderIndex int) {
	f.orderIndex = orderIndex
	f.updatedAt = time.Now()
}

func (f *FAQ) SetPublished(isPublished bool) {
	f.isPublished = isPublished
	f.updatedAt = time.Now()
}

func validateQuestion(question string) []string {
	if strings.TrimSpace(question) == "" {
		return []string{"question is required"}
	}
	return nil
}

func validateAnswer(answer string) []string {
	if strings.TrimSpace(answer) == "" {
		return []string{"answer is required"}
	}
	return nil
}

type CreateFAQInput struct {
	OwnerType   OwnerType
	OwnerID     string
	Question    string
	Answer      string
	OrderIndex  int
	IsPublished bool
}

type UpdateFAQInput struct {
	ID          string
	Question    *string
	Answer      *string
	OrderIndex  *int
	IsPublished *bool
}

// FAQFilter drives GetByOwner — PublishedOnly is false for the admin view
// (sees drafts too) and true for the public read.
type FAQFilter struct {
	OwnerType     OwnerType
	OwnerID       string
	PublishedOnly bool
}
