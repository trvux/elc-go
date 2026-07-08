package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// Review is a customer-submitted rating/comment on a product or service,
// modeled after public review widgets like Điện Máy Xanh/CellphoneS: anyone
// can submit (no auth), it goes live immediately (IsPublished defaults to
// true), and app-side heuristics only intervene to auto-hide likely
// spam/abuse for a human to review later — see NewReview's blocklist check.
// Exactly one of ProductID/ServiceID is set (unlike Inquiry, a review must
// be about something).
type Review struct {
	id            string
	productID     *string
	serviceID     *string
	rating        int
	comment       string
	reviewerName  string
	reviewerPhone *string
	isPublished   bool
	sourceIP      *string
	userAgent     *string
	createdAt     time.Time
	updatedAt     time.Time
}

const (
	maxCommentLength      = 2000
	maxReviewerNameLength = 100
)

// NewReview validates and creates a new Review from a public form
// submission. sourceIP/userAgent are request metadata, not user input, so
// they're not validated — only recorded for later spam/abuse review.
func NewReview(
	productID, serviceID *string,
	rating int,
	comment, reviewerName string,
	reviewerPhone *string,
	sourceIP, userAgent *string,
) (*Review, error) {
	fields := map[string][]string{}

	if errs := validateSingleEntity(productID, serviceID); len(errs) > 0 {
		fields["entity"] = errs
	}
	if errs := validateRating(rating); len(errs) > 0 {
		fields["rating"] = errs
	}
	if errs := validateComment(comment); len(errs) > 0 {
		fields["comment"] = errs
	}
	if errs := validateReviewerName(reviewerName); len(errs) > 0 {
		fields["reviewerName"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &Review{
		productID:     productID,
		serviceID:     serviceID,
		rating:        rating,
		comment:       comment,
		reviewerName:  reviewerName,
		reviewerPhone: reviewerPhone,
		// Auto-filter: a submission matching the blocklist starts hidden
		// rather than rejected outright — rejecting teaches spammers which
		// words trip the filter, hiding just routes it to the moderation
		// screen (see docs on the admin "Feedback" sidebar entry) for a human
		// to confirm or restore.
		isPublished: !containsBlockedContent(comment) && !containsBlockedContent(reviewerName),
		sourceIP:    sourceIP,
		userAgent:   userAgent,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// RehydrateReview reconstructs a Review from a trusted DB row — no
// validation. Only the infrastructure layer should call this.
func RehydrateReview(
	id string,
	productID, serviceID *string,
	rating int,
	comment, reviewerName string,
	reviewerPhone *string,
	isPublished bool,
	sourceIP, userAgent *string,
	createdAt, updatedAt time.Time,
) *Review {
	return &Review{
		id:            id,
		productID:     productID,
		serviceID:     serviceID,
		rating:        rating,
		comment:       comment,
		reviewerName:  reviewerName,
		reviewerPhone: reviewerPhone,
		isPublished:   isPublished,
		sourceIP:      sourceIP,
		userAgent:     userAgent,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}
}

func (r *Review) ID() string             { return r.id }
func (r *Review) ProductID() *string     { return r.productID }
func (r *Review) ServiceID() *string     { return r.serviceID }
func (r *Review) Rating() int            { return r.rating }
func (r *Review) Comment() string        { return r.comment }
func (r *Review) ReviewerName() string   { return r.reviewerName }
func (r *Review) ReviewerPhone() *string { return r.reviewerPhone }
func (r *Review) IsPublished() bool      { return r.isPublished }
func (r *Review) SourceIP() *string      { return r.sourceIP }
func (r *Review) UserAgent() *string     { return r.userAgent }
func (r *Review) CreatedAt() time.Time   { return r.createdAt }
func (r *Review) UpdatedAt() time.Time   { return r.updatedAt }

func (r *Review) SetPublished(isPublished bool) {
	r.isPublished = isPublished
	r.updatedAt = time.Now()
}

func validateSingleEntity(productID, serviceID *string) []string {
	count := 0
	for _, id := range []*string{productID, serviceID} {
		if id != nil && *id != "" {
			count++
		}
	}
	if count != 1 {
		return []string{"exactly one of productId/serviceId must be set"}
	}
	return nil
}

func validateRating(rating int) []string {
	if rating < 1 || rating > 5 {
		return []string{"rating must be between 1 and 5"}
	}
	return nil
}

func validateComment(comment string) []string {
	var errs []string
	trimmed := strings.TrimSpace(comment)
	if trimmed == "" {
		errs = append(errs, "comment is required")
	} else if utf8.RuneCountInString(trimmed) > maxCommentLength {
		errs = append(errs, "comment must not exceed 2000 characters")
	}
	return errs
}

func validateReviewerName(name string) []string {
	var errs []string
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		errs = append(errs, "reviewerName is required")
	} else if utf8.RuneCountInString(trimmed) > maxReviewerNameLength {
		errs = append(errs, "reviewerName must not exceed 100 characters")
	}
	return errs
}

// CreateReviewInput is the public-facing create payload. IsPublished is
// deliberately absent — the submitter never controls it, see NewReview.
type CreateReviewInput struct {
	ProductID     *string
	ServiceID     *string
	Rating        int
	Comment       string
	ReviewerName  string
	ReviewerPhone *string
	SourceIP      *string
	UserAgent     *string
}

// ReviewFilter drives both the public listing (ProductID/ServiceID +
// IsPublished=&true) and the admin moderation screen (any combination,
// usually IsPublished left nil to see everything).
type ReviewFilter struct {
	ProductID   *string
	ServiceID   *string
	IsPublished *bool
	Limit       int
	Offset      int
}

// ReviewSummary aggregates a product/service's published reviews for
// display and for schema.org AggregateRating JSON-LD.
type ReviewSummary struct {
	Count   int
	Average float64
}
