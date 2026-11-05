package domain

import (
	"net/url"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// Source says where a redirect came from.
type Source string

const (
	SourceManual Source = "manual" // typed in the admin UI
	SourceAuto   Source = "auto"   // created by the slug-change trigger (migration 000002)
	SourceImport Source = "import" // moved over from the old hand-built nginx file
)

func (s Source) IsValid() bool {
	switch s {
	case SourceManual, SourceAuto, SourceImport:
		return true
	default:
		return false
	}
}

// siteHosts are treated as "this site": a full URL on them is reduced to its path.
var siteHosts = map[string]bool{"dienmayelc.com.vn": true, "www.dienmayelc.com.vn": true}

// reservedPrefixes can never be the SOURCE of a redirect: nginx consults the
// redirect map before everything else, so redirecting one of these would hide
// the admin panel, the API or static assets.
var reservedPrefixes = []string{"/admin", "/api", "/_next", "/images", "/logo", "/brands", "/fonts", "/.well-known"}

// MaxChainDepth bounds how far a redirect target is followed when flattening.
const MaxChainDepth = 10

// NormalizePath reduces a user-typed path or same-site URL to the canonical
// form stored and matched: leading slash, no query/fragment, no trailing
// slash, no duplicate slashes. Case is preserved (nginx matches case-sensitively).
func NormalizePath(raw string) (string, error) {
	p, external, err := normalize(raw)
	if err != nil {
		return "", err
	}
	if external {
		return "", validationErr("from_path", "phải là đường dẫn trên website này, ví dụ /dich-vu/ten-cu")
	}
	return p, nil
}

// NormalizeTarget is NormalizePath for the destination, which may also be an
// absolute https URL on another host (returned unchanged).
func NormalizeTarget(raw string) (string, error) {
	p, _, err := normalize(raw)
	return p, err
}

func normalize(raw string) (path string, external bool, err error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", false, validationErr("path", "không được để trống")
	}
	if strings.Contains(s, "://") {
		u, perr := url.Parse(s)
		if perr != nil || u.Host == "" {
			return "", false, validationErr("path", "URL không hợp lệ")
		}
		if !siteHosts[strings.ToLower(u.Host)] {
			if u.Scheme != "https" {
				return "", false, validationErr("path", "URL ngoài website phải dùng https")
			}
			return s, true, nil
		}
		s = u.EscapedPath()
	}
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	if decoded, derr := url.PathUnescape(s); derr == nil {
		s = decoded
	}
	if strings.ContainsAny(s, " \t\r\n") {
		return "", false, validationErr("path", "không được chứa khoảng trắng")
	}
	if !strings.HasPrefix(s, "/") {
		return "", false, validationErr("path", "phải bắt đầu bằng /")
	}
	for strings.Contains(s, "//") {
		s = strings.ReplaceAll(s, "//", "/")
	}
	if len(s) > 1 {
		s = strings.TrimRight(s, "/")
	}
	return s, false, nil
}

func validationErr(field, msg string) error {
	return apperr.NewValidationError("validation failed", map[string][]string{field: {msg}})
}

func isReserved(path string) bool {
	for _, p := range reservedPrefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

// Redirect sends one old URL permanently (HTTP 308) to a new one.
type Redirect struct {
	id        string
	fromPath  string
	toPath    string
	isActive  bool
	source    Source
	note      string
	createdAt time.Time
	updatedAt time.Time
}

// NewRedirect validates and creates a redirect from admin input.
func NewRedirect(from, to, note string, source Source) (*Redirect, error) {
	fields := map[string][]string{}

	fromPath, err := NormalizePath(from)
	if err != nil {
		fields["from_path"] = firstField(err)
	} else if fromPath == "/" {
		fields["from_path"] = []string{"không thể chuyển hướng trang chủ"}
	} else if isReserved(fromPath) {
		fields["from_path"] = []string{"không thể chuyển hướng đường dẫn hệ thống (/admin, /api, /_next, ảnh...)"}
	}

	toPath, err := NormalizeTarget(to)
	if err != nil {
		fields["to_path"] = firstField(err)
	}

	if len(fields) == 0 && fromPath == toPath {
		fields["to_path"] = []string{"đích trùng với đường dẫn nguồn"}
	}
	if !source.IsValid() {
		fields["source"] = []string{"nguồn không hợp lệ"}
	}
	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &Redirect{
		fromPath: fromPath, toPath: toPath, isActive: true, source: source,
		note: strings.TrimSpace(note), createdAt: now, updatedAt: now,
	}, nil
}

// firstField pulls the message list out of a single-field validation error.
func firstField(err error) []string {
	if ae, ok := err.(*apperr.AppError); ok {
		for _, msgs := range ae.Fields {
			return msgs
		}
	}
	return []string{err.Error()}
}

// RehydrateRedirect reconstructs from a trusted DB row — no validation.
func RehydrateRedirect(id, fromPath, toPath string, isActive bool, source Source, note string, createdAt, updatedAt time.Time) *Redirect {
	return &Redirect{id: id, fromPath: fromPath, toPath: toPath, isActive: isActive, source: source, note: note, createdAt: createdAt, updatedAt: updatedAt}
}

func (r *Redirect) ID() string           { return r.id }
func (r *Redirect) FromPath() string     { return r.fromPath }
func (r *Redirect) ToPath() string       { return r.toPath }
func (r *Redirect) IsActive() bool       { return r.isActive }
func (r *Redirect) Source() Source       { return r.source }
func (r *Redirect) Note() string         { return r.note }
func (r *Redirect) CreatedAt() time.Time { return r.createdAt }
func (r *Redirect) UpdatedAt() time.Time { return r.updatedAt }

// Retarget changes the destination. The caller is responsible for loop checks.
func (r *Redirect) Retarget(to string) error {
	toPath, err := NormalizeTarget(to)
	if err != nil {
		return apperr.NewValidationError("validation failed", map[string][]string{"to_path": firstField(err)})
	}
	if toPath == r.fromPath {
		return validationErr("to_path", "đích trùng với đường dẫn nguồn")
	}
	r.toPath = toPath
	r.updatedAt = time.Now()
	return nil
}

func (r *Redirect) SetNote(note string) {
	r.note = strings.TrimSpace(note)
	r.updatedAt = time.Now()
}

func (r *Redirect) SetActive(active bool) {
	r.isActive = active
	r.updatedAt = time.Now()
}

type CreateRedirectInput struct {
	FromPath string
	ToPath   string
	Note     string
}

type UpdateRedirectInput struct {
	ID       string
	ToPath   *string
	Note     *string
	IsActive *bool
}

// Filter drives List. Limit <= 0 means the default page size.
type Filter struct {
	Search string // matches from_path, to_path or note
	Source Source // empty = every source
	Limit  int
	Offset int
}
