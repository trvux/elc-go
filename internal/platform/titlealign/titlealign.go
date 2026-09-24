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

// OrDefault returns Left when v is empty (the "not specified" case every
// caller treats as today's fixed behavior), otherwise v unchanged. Callers
// must still call Valid on the result — OrDefault does not validate.
func OrDefault(v string) string {
	if v == "" {
		return Left
	}
	return v
}
