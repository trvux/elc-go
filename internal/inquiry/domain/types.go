package domain

import (
	"encoding/json"
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

// LeadType is which of the lead-capture form's 3 branches (or none) the
// visitor went through. Recorded independently of ProductID/ProjectID/
// ServiceID because a visitor can pick "Bỏ qua, chỉ cần tư vấn chung" at
// every catalog picker and still tell us which branch they were in — admin
// needs to filter/prioritize by branch regardless of whether a concrete
// catalog entity ended up linked.
type LeadType string

const (
	LeadTypeProduct LeadType = "product"
	LeadTypeService LeadType = "service"
	LeadTypeProject LeadType = "project"
	LeadTypeGeneral LeadType = "general"
)

func (t LeadType) IsValid() bool {
	switch t {
	case LeadTypeProduct, LeadTypeService, LeadTypeProject, LeadTypeGeneral:
		return true
	default:
		return false
	}
}

// ContactChannel is HOW the visitor reached out — independent of LeadType,
// which is WHAT they were looking at (product/service/project/general).
// Same "what vs how" split reasoning as LeadType/SubType above. "form" is
// the only channel that captures name/phone up front, at submission time;
// the other 3 are recorded the moment the visitor clicks a Zalo/Messenger/
// Hotline link (see internal/inquiry/presentation's POST /inquiries/clicks)
// — staff fill in identity later, once they actually talk to the customer.
type ContactChannel string

const (
	ChannelForm      ContactChannel = "form"
	ChannelZalo      ContactChannel = "zalo"
	ChannelMessenger ContactChannel = "messenger"
	ChannelHotline   ContactChannel = "hotline"
)

func (c ContactChannel) IsValid() bool {
	switch c {
	case ChannelForm, ChannelZalo, ChannelMessenger, ChannelHotline:
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
	leadType     LeadType
	subType      *string
	qualifyData  json.RawMessage
	attachments  []string
	channel      ContactChannel
	status       InquiryStatus
	internalNote *string
	// conversionValue/adsConversionSyncedAt are set later, when staff marks
	// the lead 'converted' and the outcome is pushed to GA4/Ads — see
	// internal/inquiry/application/update_inquiry_status.go.
	conversionValue       *float64
	adsConversionSyncedAt *time.Time
	sourceIP              *string
	userAgent             *string
	// gclid/utm*/gaClientID are captured at creation time from the
	// visitor's own attribution cookie (see elc-temp's middleware.ts) — the
	// only way to later tie a real sale back to the ad/campaign that
	// produced it.
	gclid       *string
	utmSource   *string
	utmMedium   *string
	utmCampaign *string
	utmTerm     *string
	utmContent  *string
	gaClientID  *string
	// sessionID (the visitor's own first-party session cookie) is only
	// populated for click-origin leads — lets POST /inquiries/clicks find
	// and refresh this row instead of creating a duplicate one on a repeat
	// click. See RefreshClickContext.
	sessionID *string
	createdAt time.Time
	updatedAt time.Time
}

// NewInquiry validates and creates a new Inquiry from a public form
// submission. sourceIP/userAgent are request metadata, not user input, so
// they're not validated — only recorded for later spam/abuse review.
func NewInquiry(
	name, phone string,
	email, message *string,
	productID, projectID, serviceID *string,
	leadType LeadType, subType *string, qualifyData json.RawMessage, attachments []string,
	channel ContactChannel,
	gclid, utmSource, utmMedium, utmCampaign, utmTerm, utmContent, gaClientID, sessionID *string,
	sourceIP, userAgent *string,
) (*Inquiry, error) {
	fields := map[string][]string{}

	if channel == "" {
		channel = ChannelForm
	}
	if !channel.IsValid() {
		fields["channel"] = []string{"invalid channel"}
	}

	// Only the on-site form captures identity up front — a click-origin
	// lead (Zalo/Messenger/Hotline) has no name/phone yet, staff fill those
	// in later once they actually talk to the customer.
	requireIdentity := channel == ChannelForm
	if errs := validateName(name, requireIdentity); len(errs) > 0 {
		fields["name"] = errs
	}
	if errs := validatePhone(phone, requireIdentity); len(errs) > 0 {
		fields["phone"] = errs
	}
	if errs := validateEmail(email); len(errs) > 0 {
		fields["email"] = errs
	}
	if errs := validateSingleEntity(productID, projectID, serviceID); len(errs) > 0 {
		fields["entity"] = errs
	}
	if leadType == "" {
		leadType = LeadTypeGeneral
	}
	if !leadType.IsValid() {
		fields["leadType"] = []string{"invalid lead type"}
	}
	if errs := validateAttachments(attachments); len(errs) > 0 {
		fields["attachments"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	if qualifyData == nil {
		qualifyData = json.RawMessage("{}")
	}

	now := time.Now()
	return &Inquiry{
		name:        name,
		phone:       phone,
		email:       email,
		message:     message,
		productID:   productID,
		projectID:   projectID,
		serviceID:   serviceID,
		leadType:    leadType,
		subType:     subType,
		qualifyData: qualifyData,
		attachments: attachments,
		channel:     channel,
		gclid:       gclid,
		utmSource:   utmSource,
		utmMedium:   utmMedium,
		utmCampaign: utmCampaign,
		utmTerm:     utmTerm,
		utmContent:  utmContent,
		gaClientID:  gaClientID,
		sessionID:   sessionID,
		status:      InquiryStatusNew,
		sourceIP:    sourceIP,
		userAgent:   userAgent,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// RehydrateInquiry reconstructs an Inquiry from a trusted DB row — no
// validation. Only the infrastructure layer should call this.
func RehydrateInquiry(
	id, name, phone string,
	email, message *string,
	productID, projectID, serviceID *string,
	leadType LeadType, subType *string, qualifyData json.RawMessage, attachments []string,
	channel ContactChannel,
	gclid, utmSource, utmMedium, utmCampaign, utmTerm, utmContent, gaClientID, sessionID *string,
	status InquiryStatus,
	internalNote *string,
	conversionValue *float64, adsConversionSyncedAt *time.Time,
	sourceIP, userAgent *string,
	createdAt, updatedAt time.Time,
) *Inquiry {
	return &Inquiry{
		id:                    id,
		name:                  name,
		phone:                 phone,
		email:                 email,
		message:               message,
		productID:             productID,
		projectID:             projectID,
		serviceID:             serviceID,
		leadType:              leadType,
		subType:               subType,
		qualifyData:           qualifyData,
		attachments:           attachments,
		channel:               channel,
		gclid:                 gclid,
		utmSource:             utmSource,
		utmMedium:             utmMedium,
		utmCampaign:           utmCampaign,
		utmTerm:               utmTerm,
		utmContent:            utmContent,
		gaClientID:            gaClientID,
		sessionID:             sessionID,
		status:                status,
		internalNote:          internalNote,
		conversionValue:       conversionValue,
		adsConversionSyncedAt: adsConversionSyncedAt,
		sourceIP:              sourceIP,
		userAgent:             userAgent,
		createdAt:             createdAt,
		updatedAt:             updatedAt,
	}
}

func (i *Inquiry) ID() string                   { return i.id }
func (i *Inquiry) Name() string                 { return i.name }
func (i *Inquiry) Phone() string                { return i.phone }
func (i *Inquiry) Email() *string               { return i.email }
func (i *Inquiry) Message() *string             { return i.message }
func (i *Inquiry) ProductID() *string           { return i.productID }
func (i *Inquiry) ProjectID() *string           { return i.projectID }
func (i *Inquiry) ServiceID() *string           { return i.serviceID }
func (i *Inquiry) LeadType() LeadType           { return i.leadType }
func (i *Inquiry) SubType() *string             { return i.subType }
func (i *Inquiry) QualifyData() json.RawMessage { return i.qualifyData }
func (i *Inquiry) Attachments() []string        { return i.attachments }
func (i *Inquiry) Status() InquiryStatus        { return i.status }
func (i *Inquiry) InternalNote() *string        { return i.internalNote }
func (i *Inquiry) SourceIP() *string            { return i.sourceIP }
func (i *Inquiry) UserAgent() *string           { return i.userAgent }
func (i *Inquiry) CreatedAt() time.Time         { return i.createdAt }
func (i *Inquiry) UpdatedAt() time.Time         { return i.updatedAt }

func (i *Inquiry) Channel() ContactChannel           { return i.channel }
func (i *Inquiry) GCLID() *string                    { return i.gclid }
func (i *Inquiry) UTMSource() *string                { return i.utmSource }
func (i *Inquiry) UTMMedium() *string                { return i.utmMedium }
func (i *Inquiry) UTMCampaign() *string              { return i.utmCampaign }
func (i *Inquiry) UTMTerm() *string                  { return i.utmTerm }
func (i *Inquiry) UTMContent() *string               { return i.utmContent }
func (i *Inquiry) GAClientID() *string               { return i.gaClientID }
func (i *Inquiry) ConversionValue() *float64         { return i.conversionValue }
func (i *Inquiry) AdsConversionSyncedAt() *time.Time { return i.adsConversionSyncedAt }
func (i *Inquiry) SessionID() *string                { return i.sessionID }

// RefreshClickContext updates a click-origin inquiry with the latest touch
// — called by RecordContactClick when the same visitor (session+channel)
// clicks a contact link again before this lead is picked up, instead of
// creating a duplicate row. "Last Google-Ads-touch": entity/lead-type/
// attribution are overwritten with the newest click's values. Deliberately
// never touches name/phone/status — those belong to staff once they've
// actually talked to the customer (see MarkContacted/MarkConverted/Close).
func (i *Inquiry) RefreshClickContext(
	leadType LeadType, subType *string,
	productID, projectID, serviceID *string,
	qualifyData json.RawMessage,
	gclid, utmSource, utmMedium, utmCampaign, utmTerm, utmContent, gaClientID *string,
) {
	if qualifyData == nil {
		qualifyData = json.RawMessage("{}")
	}

	i.leadType = leadType
	i.subType = subType
	i.productID = productID
	i.projectID = projectID
	i.serviceID = serviceID
	i.qualifyData = qualifyData
	i.gclid = gclid
	i.utmSource = utmSource
	i.utmMedium = utmMedium
	i.utmCampaign = utmCampaign
	i.utmTerm = utmTerm
	i.utmContent = utmContent
	i.gaClientID = gaClientID
	i.updatedAt = time.Now()
}

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

// validateName only requires a non-empty name when requireIdentity is true
// (channel == ChannelForm) — a click-origin lead has no name yet. The
// length cap still applies whenever a name IS given, regardless of channel.
func validateName(name string, requireIdentity bool) []string {
	var errs []string
	if strings.TrimSpace(name) == "" {
		if requireIdentity {
			errs = append(errs, "name is required")
		}
		return errs
	}
	if utf8.RuneCountInString(name) > 255 {
		errs = append(errs, "name must not exceed 255 characters")
	}
	return errs
}

// validatePhone mirrors validateName's requireIdentity behavior.
func validatePhone(phone string, requireIdentity bool) []string {
	var errs []string
	if strings.TrimSpace(phone) == "" {
		if requireIdentity {
			errs = append(errs, "phone is required")
		}
		return errs
	}
	if utf8.RuneCountInString(phone) > 30 {
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

// validateAttachments caps the lead-form's photo upload step at 6 images —
// matches the limit enforced client-side in createInquirySchema (Zod), kept
// here too since the public endpoint can't rely on the client alone.
func validateAttachments(attachments []string) []string {
	if len(attachments) > 6 {
		return []string{"at most 6 attachments allowed"}
	}
	for _, a := range attachments {
		if strings.TrimSpace(a) == "" {
			return []string{"attachment url must not be empty"}
		}
	}
	return nil
}

// CreateInquiryInput is the public-facing create payload.
type CreateInquiryInput struct {
	Name        string
	Phone       string
	Email       *string
	Message     *string
	ProductID   *string
	ProjectID   *string
	ServiceID   *string
	LeadType    LeadType
	SubType     *string
	QualifyData json.RawMessage
	Attachments []string
	Channel     ContactChannel
	GCLID       *string
	UTMSource   *string
	UTMMedium   *string
	UTMCampaign *string
	UTMTerm     *string
	UTMContent  *string
	GAClientID  *string
	SourceIP    *string
	UserAgent   *string
}

// CreateContactClickInput is the public-facing payload for a Zalo/
// Messenger/Hotline click — see POST /inquiries/clicks. No Name/Phone: a
// click alone never carries the visitor's identity, unlike the on-site
// form (see requireIdentity in NewInquiry).
type CreateContactClickInput struct {
	Channel     ContactChannel
	ProductID   *string
	ProjectID   *string
	ServiceID   *string
	LeadType    LeadType
	SubType     *string
	QualifyData json.RawMessage
	SessionID   *string
	GCLID       *string
	UTMSource   *string
	UTMMedium   *string
	UTMCampaign *string
	UTMTerm     *string
	UTMContent  *string
	GAClientID  *string
	SourceIP    *string
	UserAgent   *string
}

// InquiryFilter drives the admin list view.
type InquiryFilter struct {
	Status string
	Search string
	Limit  int
	Offset int
}
