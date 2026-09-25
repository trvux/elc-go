package presentation

import (
	"encoding/json"
	"time"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

// inquiryResponse is what the admin panel sees over HTTP — separate from
// domain.Inquiry so the entity's internal shape can evolve independently.
type inquiryResponse struct {
	ID                    string                 `json:"id"`
	Name                  string                 `json:"name"`
	Phone                 string                 `json:"phone"`
	Email                 *string                `json:"email"`
	Message               *string                `json:"message"`
	ProductID             *string                `json:"product_id"`
	ProjectID             *string                `json:"project_id"`
	ServiceID             *string                `json:"service_id"`
	LeadType              string                 `json:"lead_type"`
	SubType               *string                `json:"sub_type"`
	QualifyData           json.RawMessage        `json:"qualify_data"`
	InterestHistory       []domain.InterestTouch `json:"interest_history"`
	Attachments           []string               `json:"attachments"`
	Channel               string                 `json:"channel"`
	GCLID                 *string                `json:"gclid"`
	UTMSource             *string                `json:"utm_source"`
	UTMMedium             *string                `json:"utm_medium"`
	UTMCampaign           *string                `json:"utm_campaign"`
	UTMTerm               *string                `json:"utm_term"`
	UTMContent            *string                `json:"utm_content"`
	GAClientID            *string                `json:"ga_client_id"`
	Status                string                 `json:"status"`
	InternalNote          *string                `json:"internal_note"`
	ConversionValue       *float64               `json:"conversion_value"`
	AdsConversionSyncedAt *string                `json:"ads_conversion_synced_at"`
	CreatedAt             string                 `json:"created_at"`
	UpdatedAt             string                 `json:"updated_at"`
}

func toInquiryResponse(i *domain.Inquiry) inquiryResponse {
	cleanQualifyData, interestHistory := splitInterestHistory(i.QualifyData())
	return inquiryResponse{
		ID:                    i.ID(),
		Name:                  i.Name(),
		Phone:                 i.Phone(),
		Email:                 i.Email(),
		Message:               i.Message(),
		ProductID:             i.ProductID(),
		ProjectID:             i.ProjectID(),
		ServiceID:             i.ServiceID(),
		LeadType:              string(i.LeadType()),
		SubType:               i.SubType(),
		QualifyData:           cleanQualifyData,
		InterestHistory:       interestHistory,
		Attachments:           i.Attachments(),
		Channel:               string(i.Channel()),
		GCLID:                 i.GCLID(),
		UTMSource:             i.UTMSource(),
		UTMMedium:             i.UTMMedium(),
		UTMCampaign:           i.UTMCampaign(),
		UTMTerm:               i.UTMTerm(),
		UTMContent:            i.UTMContent(),
		GAClientID:            i.GAClientID(),
		Status:                string(i.Status()),
		InternalNote:          i.InternalNote(),
		ConversionValue:       i.ConversionValue(),
		AdsConversionSyncedAt: formatOptionalTime(i.AdsConversionSyncedAt()),
		CreatedAt:             i.CreatedAt().Format(timeFormat),
		UpdatedAt:             i.UpdatedAt().Format(timeFormat),
	}
}

// splitInterestHistory pulls the "interestHistory" key (see
// domain.AppendInterestTouch) out of a click-origin lead's qualify_data and
// returns it as its own typed field, plus the remainder unchanged — the
// admin's generic "Thông tin khảo sát" key/value list isn't built to render
// an array, and this gets its own dedicated timeline UI instead. A blank/
// nil raw or one with no interestHistory key returns it as-is with a nil
// slice — every other qualify_data shape (form-origin leads' choice-step
// answers) round-trips untouched.
func splitInterestHistory(raw json.RawMessage) (json.RawMessage, []domain.InterestTouch) {
	if len(raw) == 0 {
		return raw, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return raw, nil
	}

	historyRaw, ok := fields["interestHistory"]
	if !ok {
		return raw, nil
	}

	var history []domain.InterestTouch
	_ = json.Unmarshal(historyRaw, &history)

	delete(fields, "interestHistory")
	cleaned, err := json.Marshal(fields)
	if err != nil {
		return raw, history
	}
	return cleaned, history
}

func formatOptionalTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format(timeFormat)
	return &formatted
}

func toInquiryResponseList(inquiries []*domain.Inquiry) []inquiryResponse {
	result := make([]inquiryResponse, len(inquiries))
	for idx, i := range inquiries {
		result[idx] = toInquiryResponse(i)
	}
	return result
}

const timeFormat = "2006-01-02T15:04:05Z07:00"

// createInquiryRequest is the public form payload. Website is a honeypot —
// a hidden field real visitors never see or fill; a bot that fills every
// field on the page trips it. See InquiryHandler.Create.
type createInquiryRequest struct {
	Name        string          `json:"name"`
	Phone       string          `json:"phone"`
	Email       *string         `json:"email"`
	Message     *string         `json:"message"`
	ProductID   *string         `json:"product_id"`
	ProjectID   *string         `json:"project_id"`
	ServiceID   *string         `json:"service_id"`
	LeadType    string          `json:"lead_type"`
	SubType     *string         `json:"sub_type"`
	QualifyData json.RawMessage `json:"qualify_data"`
	Attachments []string        `json:"attachments"`
	Channel     string          `json:"channel"`
	GCLID       *string         `json:"gclid"`
	UTMSource   *string         `json:"utm_source"`
	UTMMedium   *string         `json:"utm_medium"`
	UTMCampaign *string         `json:"utm_campaign"`
	UTMTerm     *string         `json:"utm_term"`
	UTMContent  *string         `json:"utm_content"`
	GAClientID  *string         `json:"ga_client_id"`
	Website     string          `json:"website"`
}

type updateInquiryStatusRequest struct {
	Status       string  `json:"status"`
	InternalNote *string `json:"internal_note"`
}

// updateInquiryDetailsRequest is the admin-only payload for
// InquiryHandler.UpdateDetails (PATCH /inquiries/{id}). Name/Phone are
// plain strings, not *string — the admin dialog always sends its full
// current draft (same "always send the whole form" posture as
// updateInquiryStatusRequest's Status), so there's no "omitted" case to
// distinguish from "cleared".
type updateInquiryDetailsRequest struct {
	Name            string   `json:"name"`
	Phone           string   `json:"phone"`
	ConversionValue *float64 `json:"conversion_value"`
}

// createClickRequest is the public payload for a Zalo/Messenger/Hotline
// contact-link click — see InquiryHandler.CreateClick. No honeypot: unlike
// createInquiryRequest, every field here is app-controlled/enumerated, the
// visitor never types anything into this request.
type createClickRequest struct {
	Channel   string  `json:"channel"`
	ProductID *string `json:"product_id"`
	ProjectID *string `json:"project_id"`
	ServiceID *string `json:"service_id"`
	LeadType  string  `json:"lead_type"`
	SubType   *string `json:"sub_type"`
	// PagePath isn't its own inquiries column — folded into qualify_data
	// (buildClickQualifyData) alongside form leads' choice-step answers,
	// same flexible JSONB bag, one less migration for a single debug field.
	PagePath *string `json:"page_path"`
	// The specific product/service/project name, sent by the client (it
	// already has it in hand — productName/title prop — at click time). A
	// snapshot at click time, not a live lookup by ProductID/ServiceID/
	// ProjectID: this module has no dependency on those modules' repos, and
	// what the customer actually saw named when they clicked is the more
	// honest record anyway, even if the entity gets renamed later.
	EntityName  *string `json:"entity_name"`
	SessionID   *string `json:"session_id"`
	GCLID       *string `json:"gclid"`
	UTMSource   *string `json:"utm_source"`
	UTMMedium   *string `json:"utm_medium"`
	UTMCampaign *string `json:"utm_campaign"`
	UTMTerm     *string `json:"utm_term"`
	UTMContent  *string `json:"utm_content"`
	GAClientID  *string `json:"ga_client_id"`
	// Reverse-geocoded "Phường X, Quận Y, Thành phố Z" from the browser's
	// Geolocation API, resolved server-side in elc-temp (proxy has
	// GEOCODING_API_KEY, this module never talks to Google directly) — nil
	// whenever the visitor denied/ignored the permission prompt or the
	// lookup failed, same best-effort posture as ga_client_id.
	Location *string `json:"location"`
}

func buildClickQualifyData(pagePath, entityName, location *string) json.RawMessage {
	fields := map[string]string{}
	if pagePath != nil && *pagePath != "" {
		fields["pagePath"] = *pagePath
	}
	if entityName != nil && *entityName != "" {
		fields["entityName"] = *entityName
	}
	if location != nil && *location != "" {
		fields["location"] = *location
	}
	if len(fields) == 0 {
		return json.RawMessage("{}")
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		// Marshaling plain string fields cannot realistically fail — fall
		// back to an empty object rather than propagate an error for
		// what's ultimately a nice-to-have debug field.
		return json.RawMessage("{}")
	}
	return encoded
}
