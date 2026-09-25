// Package titlealign holds the left/center/right title-alignment enum
// shared by every content module whose title/name renders as the page's H1
// (news, project, page, service, product, branch) — see platform/seo for
// the identical "one small cross-module primitive, one platform package"
// precedent. Before this, each module defined its own copy of
// TitleAlignLeft/Center/Right + a validTitleAlign switch.
package titlealign

const (
	Left   = "left"
	Center = "center"
	Right  = "right"
)

// Valid reports whether v is one of Left/Center/Right.
func Valid(v string) bool {
	switch v {
	case Left, Center, Right:
		return true
	default:
		return false
	}
}

// OrDefault returns Center when v is empty (the "not specified" case),
// otherwise v unchanged. Callers must still call Valid on the result —
// OrDefault does not validate.
//
// Was Left until 2026-09-25: many already-published articles/titles that
// predate this feature (created back when every title was hardcoded
// left-aligned, before titleAlign existed at all) read as visually "off"
// once every other page in the same section is deliberately centered —
// readers assumed it was a layout bug, not a per-record choice. Center is
// now the fallback everywhere titleAlign isn't explicitly set. Existing
// rows already stored as 'left' are a separate, explicit choice (not "not
// specified") and are handled by a one-time data backfill instead — see
// each module's own migration.
func OrDefault(v string) string {
	if v == "" {
		return Center
	}
	return v
}
