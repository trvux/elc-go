package presentation

import "github.com/trvux/elc-go/internal/inquiry/domain"

// inquiryResponse is what the admin panel sees over HTTP — separate from
// domain.Inquiry so the entity's internal shape can evolve independently.
type inquiryResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Phone        string  `json:"phone"`
	Email        *string `json:"email"`
	Message      *string `json:"message"`
	ProductID    *string `json:"product_id"`
	ProjectID    *string `json:"project_id"`
	ServiceID    *string `json:"service_id"`
	Status       string  `json:"status"`
	InternalNote *string `json:"internal_note"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func toInquiryResponse(i *domain.Inquiry) inquiryResponse {
	return inquiryResponse{
		ID:           i.ID(),
		Name:         i.Name(),
		Phone:        i.Phone(),
		Email:        i.Email(),
		Message:      i.Message(),
		ProductID:    i.ProductID(),
		ProjectID:    i.ProjectID(),
		ServiceID:    i.ServiceID(),
		Status:       string(i.Status()),
		InternalNote: i.InternalNote(),
		CreatedAt:    i.CreatedAt().Format(timeFormat),
		UpdatedAt:    i.UpdatedAt().Format(timeFormat),
	}
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
	Name      string  `json:"name"`
	Phone     string  `json:"phone"`
	Email     *string `json:"email"`
	Message   *string `json:"message"`
	ProductID *string `json:"product_id"`
	ProjectID *string `json:"project_id"`
	ServiceID *string `json:"service_id"`
	Website   string  `json:"website"`
}

type updateInquiryStatusRequest struct {
	Status       string  `json:"status"`
	InternalNote *string `json:"internal_note"`
}
