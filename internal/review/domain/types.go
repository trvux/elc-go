package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// Review is a public star rating + comment left by a site visitor about a
// product, project, service, or news article. Exactly one of ProductID/
// ProjectID/ServiceID/NewsID is set (unlike Inquiry, which allows at most
// one — a review must be about something) — which one tells you what the
// review is for, so no separate "entity type" column is stored. Publishes
// immediately (IsPublished defaults true) — no moderation gate in this
// version, see NewReview.
type Review struct {
	id            string
	productID     *string
	projectID     *string
	serviceID     *string
	newsID        *string
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

// NewReview validates and creates a new Review from a public form
// submission. sourceIP/userAgent are request metadata, not user input, so
// they're not validated — only recorded for later spam/abuse review.
func NewReview(
	rating int,
	comment, reviewerName string,
	reviewerPhone *string,
	productID, projectID, serviceID, newsID *string,
	sourceIP, userAgent *string,
) (*Review, error) {
	fields := map[string][]string{}

	if errs := validateRating(rating); len(errs) > 0 {
		fields["rating"] = errs
	}
	if errs := validateComment(comment); len(errs) > 0 {
		fields["comment"] = errs
	}
	if errs := validateReviewerName(reviewerName); len(errs) > 0 {
		fields["reviewerName"] = errs
	}
	if errs := validateSingleEntity(productID, projectID, serviceID, newsID); len(errs) > 0 {
		fields["entity"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &Review{
		rating:        rating,
		comment:       comment,
		reviewerName:  reviewerName,
		reviewerPhone: reviewerPhone,
		productID:     productID,
		projectID:     projectID,
		serviceID:     serviceID,
		newsID:        newsID,
		isPublished:   true,
		sourceIP:      sourceIP,
		userAgent:     userAgent,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

// RehydrateReview reconstructs a Review from a trusted DB row — no
// validation. Only the infrastructure layer should call this.
func RehydrateReview(
	id string,
	productID, projectID, serviceID, newsID *string,
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
		projectID:     projectID,
		serviceID:     serviceID,
		newsID:        newsID,
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
func (r *Review) ProjectID() *string     { return r.projectID }
func (r *Review) ServiceID() *string     { return r.serviceID }
func (r *Review) NewsID() *string        { return r.newsID }
func (r *Review) Rating() int            { return r.rating }
func (r *Review) Comment() string        { return r.comment }
func (r *Review) ReviewerName() string   { return r.reviewerName }
func (r *Review) ReviewerPhone() *string { return r.reviewerPhone }
func (r *Review) IsPublished() bool      { return r.isPublished }
func (r *Review) SourceIP() *string      { return r.sourceIP }
func (r *Review) UserAgent() *string     { return r.userAgent }
func (r *Review) CreatedAt() time.Time   { return r.createdAt }
func (r *Review) UpdatedAt() time.Time   { return r.updatedAt }

func validateRating(rating int) []string {
	if rating < 1 || rating > 5 {
		return []string{"rating must be between 1 and 5"}
	}
	return nil
}

func validateComment(comment string) []string {
	if strings.TrimSpace(comment) == "" {
		return []string{"comment is required"}
	}
	return nil
}

func validateReviewerName(name string) []string {
	var errs []string
	if strings.TrimSpace(name) == "" {
		errs = append(errs, "reviewerName is required")
	} else if utf8.RuneCountInString(name) > 100 {
		errs = append(errs, "reviewerName must not exceed 100 characters")
	}
	return errs
}

func validateSingleEntity(productID, projectID, serviceID, newsID *string) []string {
	count := 0
	for _, id := range []*string{productID, projectID, serviceID, newsID} {
		if id != nil && *id != "" {
			count++
		}
	}
	if count != 1 {
		return []string{"exactly one of productId/projectId/serviceId/newsId must be set"}
	}
	return nil
}

// CreateReviewInput is the public-facing create payload.
type CreateReviewInput struct {
	Rating        int
	Comment       string
	ReviewerName  string
	ReviewerPhone *string
	ProductID     *string
	ProjectID     *string
	ServiceID     *string
	NewsID        *string
	SourceIP      *string
	UserAgent     *string
}

// ReviewAggregate summarizes one entity's published reviews — feeds the
// product detail page's rating summary badge and its AggregateRating
// structured-data block. Computed on the fly from the same rows GetByEntity
// returns (see application.ListPublishedReviews), not stored — review
// volume per entity is small, this isn't a list/facet-scale aggregation
// worth denormalizing.
type ReviewAggregate struct {
	Average float64
	Count   int
}

// ReviewFilter drives the staff-only admin list — pagination only, reviews
// have no workflow status to filter by (they publish immediately, see
// NewReview's doc comment).
type ReviewFilter struct {
	Limit  int
	Offset int
}

// ProductRef is a lightweight, read-only reference to a products row,
// populated by a LEFT JOIN in this module's own SQL
// (internal/review/infrastructure) rather than importing internal/product —
// same pattern internal/category uses for its own GroupRef, see
// internal/category/domain/types.go.
type ProductRef struct {
	ID   string
	Name string
	Slug string
}

// ReviewWithProduct is what the admin list (GetAll/Count) returns — Product
// is nil when the review is for a project/service/news entity instead of a
// product, or its product was hard-deleted.
type ReviewWithProduct struct {
	*Review
	Product *ProductRef
}
