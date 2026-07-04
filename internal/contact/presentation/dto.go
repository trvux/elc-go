package presentation

import "github.com/trvux/elc-go/internal/contact/domain"

// contactResponse is what the client sees over HTTP. It is intentionally
// separate from domain.Contact — the entity's private fields are exposed
// here only through its accessor methods, and this shape can evolve
// independently of the entity's internal representation.
type contactResponse struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Label      *string `json:"label"`
	Value      string  `json:"value"`
	IsActive   bool    `json:"is_active"`
	OrderIndex int     `json:"order_index"`
	Href       string  `json:"href"`
	IsExternal bool    `json:"is_external"`
}

func toContactResponse(c *domain.Contact) contactResponse {
	return contactResponse{
		ID:         c.ID(),
		Type:       c.Type(),
		Label:      c.Label(),
		Value:      c.Value(),
		IsActive:   c.IsActive(),
		OrderIndex: c.OrderIndex(),
		Href:       c.Href(),
		IsExternal: c.IsExternal(),
	}
}

func toContactResponseList(contacts []*domain.Contact) []contactResponse {
	result := make([]contactResponse, len(contacts))
	for i, c := range contacts {
		result[i] = toContactResponse(c)
	}
	return result
}

type createContactRequest struct {
	Type       string  `json:"type"`
	Label      *string `json:"label"`
	Value      string  `json:"value"`
	IsActive   bool    `json:"is_active"`
	OrderIndex int     `json:"order_index"`
}

type updateContactRequest struct {
	Type       *string `json:"type"`
	Label      *string `json:"label"`
	Value      *string `json:"value"`
	IsActive   *bool   `json:"is_active"`
	OrderIndex *int    `json:"order_index"`
}
