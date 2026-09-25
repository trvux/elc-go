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

// OrDefault returns fallback when v is empty (the "not specified" case),
// otherwise v unchanged. Callers must still call Valid on the result —
// OrDefault does not validate.
//
// The right fallback depends on how the module renders its title, not on
// the module itself:
//   - news/project/page/branch render the title as a standalone, full-width
//     hero heading (see the Linear-style article header work, 2026-09-25) —
//     Center reads better there, and many already-published titles predate
//     titleAlign entirely (created back when every title was hardcoded
//     left-aligned), so readers mistook their left alignment for a layout
//     bug once every other page in the section was deliberately centered.
//   - product/service render the title beside a price/spec sidebar in a
//     two-column detail layout (ProductDetailModule/ServiceDetailModule) —
//     centering it there looks broken (confirmed against production
//     screenshots, 2026-09-25), so Left stays the fallback for these two.
//
// Existing rows already stored as a specific value are a separate, explicit
// choice (not "not specified") and are handled by a one-time data backfill
// instead — see each module's own migration.
func OrDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
