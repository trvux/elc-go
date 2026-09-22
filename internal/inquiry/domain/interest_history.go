package domain

import (
	"encoding/json"
	"time"
)

// maxInterestHistory caps how many touches a single dedup'd click-origin
// lead accumulates — a long-lived session browsing dozens of products
// doesn't need every single one kept, just a reasonable trail for staff to
// read. Oldest entries drop off first.
const maxInterestHistory = 20

// InterestTouch is one entry in a click-origin lead's accumulated history —
// what the visitor was looking at each time they clicked a Zalo/Messenger/
// Hotline link, in order. See AppendInterestTouch's doc comment for why
// this exists instead of just overwriting on every repeat click.
type InterestTouch struct {
	EntityName string `json:"entityName,omitempty"`
	PagePath   string `json:"pagePath,omitempty"`
	At         string `json:"at"`
}

// AppendInterestTouch folds one click's qualify_data (built by
// buildClickQualifyData — just this click's pagePath/entityName) into a
// running "interestHistory" array on top of whatever qualify_data the lead
// already had, instead of RefreshClickContext's old behavior of fully
// overwriting it on every repeat click from the same session+channel.
// Staff opening a dedup'd lead previously only ever saw the visitor's LAST
// touch (e.g. "Vệ sinh bảo trì") even if they'd actually browsed 3 products
// and 2 services first — this keeps that trail.
//
// previous is the existing lead's current qualify_data (nil/empty for a
// brand new click-origin lead, which seeds a 1-entry history instead of
// appending to one). current is this click's freshly-built qualify_data.
// entityName/pagePath stay at the top level too (unchanged shape, still the
// latest touch) — only interestHistory is new.
func AppendInterestTouch(previous, current json.RawMessage) json.RawMessage {
	history := []InterestTouch{}
	if len(previous) > 0 {
		var prev struct {
			InterestHistory []InterestTouch `json:"interestHistory"`
		}
		if err := json.Unmarshal(previous, &prev); err == nil {
			history = prev.InterestHistory
		}
	}

	var curr struct {
		EntityName string `json:"entityName"`
		PagePath   string `json:"pagePath"`
	}
	_ = json.Unmarshal(current, &curr)

	history = append(history, InterestTouch{
		EntityName: curr.EntityName,
		PagePath:   curr.PagePath,
		At:         time.Now().UTC().Format(time.RFC3339),
	})
	if len(history) > maxInterestHistory {
		history = history[len(history)-maxInterestHistory:]
	}

	result := map[string]any{
		"interestHistory": history,
	}
	if curr.EntityName != "" {
		result["entityName"] = curr.EntityName
	}
	if curr.PagePath != "" {
		result["pagePath"] = curr.PagePath
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		// Marshaling this shape cannot realistically fail — fall back to
		// the current click's own data, same posture as
		// buildClickQualifyData's own fallback.
		return current
	}
	return encoded
}
