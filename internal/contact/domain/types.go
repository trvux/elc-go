package domain

import (
	"strings"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

type Contact struct {
	id          string
	contactType string
	label       *string
	value       string
	isActive    bool
	orderIndex  int
}

// NewContact validates input and creates a new Contact. Use this when creating
// a contact from user-submitted data.
func NewContact(contactType string, label *string, value string, isActive bool, orderIndex int) (*Contact, error) {
	fields := map[string][]string{}

	if errs := validateType(contactType); len(errs) > 0 {
		fields["type"] = errs
	}
	if errs := validateValue(value); len(errs) > 0 {
		fields["value"] = errs
	}
	if errs := validateLabel(label); len(errs) > 0 {
		fields["label"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	return &Contact{
		contactType: contactType,
		label:       label,
		value:       value,
		isActive:    isActive,
		orderIndex:  orderIndex,
	}, nil
}

// RehydrateContact reconstructs a Contact from trusted, already-validated data
// (a DB row). Only the infrastructure layer should call this — no validation here.
func RehydrateContact(id, contactType string, label *string, value string, isActive bool, orderIndex int) *Contact {
	return &Contact{
		id:          id,
		contactType: contactType,
		label:       label,
		value:       value,
		isActive:    isActive,
		orderIndex:  orderIndex,
	}
}

func (c *Contact) ID() string      { return c.id }
func (c *Contact) Type() string    { return c.contactType }
func (c *Contact) Label() *string  { return c.label }
func (c *Contact) Value() string   { return c.value }
func (c *Contact) IsActive() bool  { return c.isActive }
func (c *Contact) OrderIndex() int { return c.orderIndex }

// Href is computed on demand, never stored, so it can never go out of sync
// with Type/Value. Mirrors modules/contact/domain/utils.ts getContactHref.
func (c *Contact) Href() string {
	if c.value == "" {
		return ""
	}
	cleanValue := strings.ReplaceAll(c.value, " ", "")
	if strings.HasPrefix(c.value, "http") {
		return c.value
	}

	switch c.contactType {
	case "phone":
		return "tel:" + cleanValue
	case "email":
		return "mailto:" + c.value
	case "zalo":
		return "https://zalo.me/" + cleanValue
	case "messenger":
		return "https://m.me/" + c.value
	case "facebook":
		return "https://facebook.com/" + c.value
	default:
		return c.value
	}
}

func (c *Contact) IsExternal() bool {
	href := c.Href()
	return !strings.HasPrefix(href, "tel:") && !strings.HasPrefix(href, "mailto:")
}

func (c *Contact) UpdateType(contactType string) error {
	if errs := validateType(contactType); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"type": errs})
	}
	c.contactType = contactType
	return nil
}

func (c *Contact) UpdateValue(value string) error {
	if errs := validateValue(value); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"value": errs})
	}
	c.value = value
	return nil
}

func (c *Contact) UpdateLabel(label *string) error {
	if errs := validateLabel(label); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"label": errs})
	}
	c.label = label
	return nil
}

func (c *Contact) SetActive(isActive bool) {
	c.isActive = isActive
}

func (c *Contact) Reorder(orderIndex int) {
	c.orderIndex = orderIndex
}

func validateType(contactType string) []string {
	if contactType == "" {
		return []string{"type is required"}
	}
	return nil
}

func validateValue(value string) []string {
	if value == "" {
		return []string{"value is required"}
	}
	return nil
}

func validateLabel(label *string) []string {
	if label != nil && len(*label) > 100 {
		return []string{"label must be at most 100 characters"}
	}
	return nil
}

type CreateContactInput struct {
	Type       string
	Label      *string
	Value      string
	IsActive   bool
	OrderIndex int
}

type UpdateContactInput struct {
	ID         string
	Type       *string
	Label      *string
	Value      *string
	IsActive   *bool
	OrderIndex *int
}

type ContactFilter struct {
	Type   string
	Search string
}
