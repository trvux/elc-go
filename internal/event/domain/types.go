package domain

import (
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// EventName is a fixed, typed taxonomy modeled on Google Analytics 4's
// recommended events (verb_noun, e.g. `view_item`) rather than arbitrary
// strings — see the migration's doc comment for why that distinction
// matters here (the previous implementation drifted into meaningless event
// names like `scroll_50`).
type EventName string

const (
	// EventViewItem fires when a visitor views a product/project/service
	// detail page — GA4's own name for this exact action.
	EventViewItem EventName = "view_item"
	// EventGenerateLead fires when a visitor submits the inquiry form (see
	// internal/inquiry) — also GA4's own name, chosen deliberately so the
	// two modules speak the same vocabulary for a future funnel view
	// (view_item -> generate_lead conversion rate per product/category).
	EventGenerateLead EventName = "generate_lead"
)

func (n EventName) IsValid() bool {
	switch n {
	case EventViewItem, EventGenerateLead:
		return true
	default:
		return false
	}
}

// EntityType mirrors internal/inquiry's product/project/service split.
type EntityType string

const (
	EntityTypeProduct EntityType = "product"
	EntityTypeProject EntityType = "project"
	EntityTypeService EntityType = "service"
)

func (t EntityType) IsValid() bool {
	switch t {
	case EntityTypeProduct, EntityTypeProject, EntityTypeService:
		return true
	default:
		return false
	}
}

// Event is a single append-only analytics record. Deliberately no FK to
// products/projects/services (see ARCHITECTURE notes in the migration) —
// this is a high-volume, best-effort log where a dangling reference to a
// since-deleted item is still valid historical data, not a bug.
type Event struct {
	id         string
	name       EventName
	entityType *EntityType
	entityID   *string
	pagePath   *string
	sessionID  *string
	createdAt  time.Time
}

// NewEvent validates and creates a new Event from a public tracking call.
func NewEvent(name EventName, entityType *EntityType, entityID, pagePath, sessionID *string) (*Event, error) {
	fields := map[string][]string{}

	if !name.IsValid() {
		fields["name"] = []string{"invalid event name"}
	}
	if entityType != nil && !entityType.IsValid() {
		fields["entityType"] = []string{"invalid entity type"}
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	return &Event{
		name:       name,
		entityType: entityType,
		entityID:   entityID,
		pagePath:   pagePath,
		sessionID:  sessionID,
		createdAt:  time.Now(),
	}, nil
}

func RehydrateEvent(id string, name EventName, entityType *EntityType, entityID, pagePath, sessionID *string, createdAt time.Time) *Event {
	return &Event{
		id:         id,
		name:       name,
		entityType: entityType,
		entityID:   entityID,
		pagePath:   pagePath,
		sessionID:  sessionID,
		createdAt:  createdAt,
	}
}

func (e *Event) ID() string              { return e.id }
func (e *Event) Name() EventName         { return e.name }
func (e *Event) EntityType() *EntityType { return e.entityType }
func (e *Event) EntityID() *string       { return e.entityID }
func (e *Event) PagePath() *string       { return e.pagePath }
func (e *Event) SessionID() *string      { return e.sessionID }
func (e *Event) CreatedAt() time.Time    { return e.createdAt }

// CreateEventInput is the public-facing log payload.
type CreateEventInput struct {
	Name       EventName
	EntityType *EntityType
	EntityID   *string
	PagePath   *string
	SessionID  *string
}

// TopViewedFilter bounds the "most viewed" aggregation (dashboard panel).
type TopViewedFilter struct {
	EntityType EntityType
	Since      time.Time
	Limit      int
}

// EntityViewCount is one row of the "most viewed" aggregation.
type EntityViewCount struct {
	EntityID string
	Count    int
}
