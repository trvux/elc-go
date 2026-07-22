package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// ChatLogEntry records one message a shopper typed into the AI chat finder
// (see product/application/chat_search.go's ChatSearchProducts and the
// frontend's ProductChatFinder.tsx) — not the bot's reply, just what the
// shopper actually asked. Every message carries real purchase intent/pain
// points regardless of which internal path handled it (a fresh search, a
// comparison follow-up, a "loại nào rẻ nhất?" criterion question, or a
// purchase/off-topic question routed straight to Zalo) — Kind records
// which, so later analysis can tell "the shopper was comparing" from "the
// shopper wanted to buy" without re-deriving intent from the raw text.
type ChatLogEntry struct {
	id        string
	visitorID string
	message   string
	kind      string
	createdAt time.Time
}

// Mirrors the frontend's own Turn.kind (ProductChatFinder.tsx) plus the two
// "info" turn variants it distinguishes in submitMessage — kept as an
// explicit whitelist (see validChatLogKinds) so a typo or a modified client
// can't quietly pollute the data with a kind nothing downstream expects.
const (
	ChatLogKindSearch           = "search"
	ChatLogKindCompare          = "compare"
	ChatLogKindRank             = "rank"
	ChatLogKindOffTopic         = "off_topic"
	ChatLogKindPurchaseIntent   = "purchase_intent"
	ChatLogKindNoContextCompare = "no_context_compare"
)

var validChatLogKinds = map[string]bool{
	ChatLogKindSearch: true, ChatLogKindCompare: true, ChatLogKindRank: true,
	ChatLogKindOffTopic: true, ChatLogKindPurchaseIntent: true, ChatLogKindNoContextCompare: true,
}

// chatLogMessageMaxRunes bounds storage growth from a runaway/scripted
// client — a real shopper message is a sentence or two, never anywhere
// close to this.
const chatLogMessageMaxRunes = 2000

// NewChatLogEntry validates and creates a new entry from a public request.
func NewChatLogEntry(visitorID, message, kind string) (*ChatLogEntry, error) {
	fields := map[string][]string{}

	if strings.TrimSpace(visitorID) == "" {
		fields["visitor_id"] = []string{"visitor_id is required"}
	}
	if strings.TrimSpace(message) == "" {
		fields["message"] = []string{"message is required"}
	} else if utf8.RuneCountInString(message) > chatLogMessageMaxRunes {
		fields["message"] = []string{fmt.Sprintf("message must not exceed %d characters", chatLogMessageMaxRunes)}
	}
	if !validChatLogKinds[kind] {
		fields["kind"] = []string{"kind must be one of: search, compare, rank, off_topic, purchase_intent, no_context_compare"}
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	return &ChatLogEntry{
		visitorID: visitorID,
		message:   message,
		kind:      kind,
		createdAt: time.Now(),
	}, nil
}

// RehydrateChatLogEntry reconstructs an entry from a trusted DB row — no
// validation. Only the infrastructure layer should call this.
func RehydrateChatLogEntry(id, visitorID, message, kind string, createdAt time.Time) *ChatLogEntry {
	return &ChatLogEntry{id: id, visitorID: visitorID, message: message, kind: kind, createdAt: createdAt}
}

func (e *ChatLogEntry) ID() string           { return e.id }
func (e *ChatLogEntry) VisitorID() string    { return e.visitorID }
func (e *ChatLogEntry) Message() string      { return e.message }
func (e *ChatLogEntry) Kind() string         { return e.kind }
func (e *ChatLogEntry) CreatedAt() time.Time { return e.createdAt }

// ChatLogFilter drives the admin list view.
type ChatLogFilter struct {
	Kind   string
	Search string
	Limit  int
	Offset int
}
