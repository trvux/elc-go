package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// InquiryStatus is the lead's position in the manual follow-up pipeline.
// Mirrors the auth module's UserStatus shape: a private field, explicit
// constants, defaulted in the constructor, mutated only via named methods.
type InquiryStatus string

const (
	InquiryStatusNew       InquiryStatus = "new"
	InquiryStatusContacted InquiryStatus = "contacted"
	InquiryStatusConverted InquiryStatus = "converted"
	InquiryStatusClosed    InquiryStatus = "closed"
)

func (s InquiryStatus) IsValid() bool {
	switch s {
	case InquiryStatusNew, InquiryStatusContacted, InquiryStatusConverted, InquiryStatusClosed:
		return true
	default:
		return false
	}
}

// Inquiry is a customer-submitted request for consultation/quote from the
// public site. At most one of ProductID/ProjectID/ServiceID is set — which
// one tells you what the customer was looking at, so no separate "source
// type" field is stored (computed on demand if a display label is needed,
// same reasoning as Contact.Href() in internal/contact/domain/types.go).
type Inquiry struct {
	id           string
	name         string
	phone        string
	email        *string
	message      *string
	productID    *string
	projectID    *string
	serviceID    *string
	status       InquiryStatus
	internalNote *string
	sourceIP     *string
	userAgent    *string
	createdAt    time.Time
	updatedAt    time.Time
}

// NewInquiry validates and creates a new Inquiry from a public form
// submission. sourceIP/userAgent are request metadata, not user input, so
// they're not validated — only recorded for later spam/abuse review.
func NewInquiry(
	name, phone string,
	email, message *string,
	productID, projectID, serviceID *string,
	sourceIP, userAgent *string,
) (*Inquiry, error) {
	fields := map[string][]string{}

	if errs := validateName(name); len(errs) > 0 {
		fields["name"] = errs
	}
	if errs := validatePhone(phone); len(errs) > 0 {
		fields["phone"] = errs
	}
	if errs := validateEmail(email); len(errs) > 0 {
		fields["email"] = errs
	}
	if errs := validateSingleEntity(productID, projectID, serviceID); len(errs) > 0 {
		fields["entity"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &Inquiry{
		name:      name,
		phone:     phone,
		email:     email,
		message:   message,
		productID: productID,
		projectID: projectID,
		serviceID: serviceID,
		status:    InquiryStatusNew,
		sourceIP:  sourceIP,
		userAgent: userAgent,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// RehydrateInquiry reconstructs an Inquiry from a trusted DB row — no
// validation. Only the infrastructure layer should call this.
func RehydrateInquiry(
	id, name, phone string,
	email, message *string,
	productID, projectID, serviceID *string,
	status InquiryStatus,
	internalNote, sourceIP, userAgent *string,
	createdAt, updatedAt time.Time,
) *Inquiry {
	return &Inquiry{
		id:           id,
		name:         name,
		phone:        phone,
		email:        email,
		message:      message,
		productID:    productID,
		projectID:    projectID,
		serviceID:    serviceID,
		status:       status,
		internalNote: internalNote,
		sourceIP:     sourceIP,
		userAgent:    userAgent,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

func (i *Inquiry) ID() string            { return i.id }
func (i *Inquiry) Name() string          { return i.name }
func (i *Inquiry) Phone() string         { return i.phone }
func (i *Inquiry) Email() *string        { return i.email }
func (i *Inquiry) Message() *string      { return i.message }
func (i *Inquiry) ProductID() *string    { return i.productID }
func (i *Inquiry) ProjectID() *string    { return i.projectID }
func (i *Inquiry) ServiceID() *string    { return i.serviceID }
func (i *Inquiry) Status() InquiryStatus { return i.status }
func (i *Inquiry) InternalNote() *string { return i.internalNote }
func (i *Inquiry) SourceIP() *string     { return i.sourceIP }
func (i *Inquiry) UserAgent() *string    { return i.userAgent }
func (i *Inquiry) CreatedAt() time.Time  { return i.createdAt }
func (i *Inquiry) UpdatedAt() time.Time  { return i.updatedAt }

func (i *Inquiry) MarkContacted() {
	i.status = InquiryStatusContacted
	i.updatedAt = time.Now()
}

func (i *Inquiry) MarkConverted() {
	i.status = InquiryStatusConverted
	i.updatedAt = time.Now()
}

func (i *Inquiry) Close() {
	i.status = InquiryStatusClosed
	i.updatedAt = time.Now()
}

func (i *Inquiry) Reopen() {
	i.status = InquiryStatusNew
	i.updatedAt = time.Now()
}

func (i *Inquiry) SetInternalNote(note *string) {
	i.internalNote = note
	i.updatedAt = time.Now()
}

func validateName(name string) []string {
	var errs []string
	if strings.TrimSpace(name) == "" {
		errs = append(errs, "name is required")
	} else if utf8.RuneCountInString(name) > 255 {
		errs = append(errs, "name must not exceed 255 characters")
	}
	return errs
}

func validatePhone(phone string) []string {
	var errs []string
	if strings.TrimSpace(phone) == "" {
		errs = append(errs, "phone is required")
	} else if utf8.RuneCountInString(phone) > 30 {
		errs = append(errs, "phone must not exceed 30 characters")
	}
	return errs
}

func validateEmail(email *string) []string {
	if email == nil || *email == "" {
		return nil
	}
	if !strings.Contains(*email, "@") || !strings.Contains(*email, ".") {
		return []string{"email is not valid"}
	}
	return nil
}

func validateSingleEntity(productID, projectID, serviceID *string) []string {
	count := 0
	for _, id := range []*string{productID, projectID, serviceID} {
		if id != nil && *id != "" {
			count++
		}
	}
	if count > 1 {
		return []string{"at most one of productId/projectId/serviceId may be set"}
	}
	return nil
}

// CreateInquiryInput is the public-facing create payload.
type CreateInquiryInput struct {
	Name      string
	Phone     string
	Email     *string
	Message   *string
	ProductID *string
	ProjectID *string
	ServiceID *string
	SourceIP  *string
	UserAgent *string
}

// InquiryFilter drives the admin list view.
type InquiryFilter struct {
	Status string
	Search string
	Limit  int
	Offset int
}
