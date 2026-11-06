package domain

import (
	"bytes"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Attachment is one file a visitor sent with a lead. Two shapes live in the database:
//
//   - legacy: a plain https URL of a public photo (leads before private files existed),
//     stored as a bare JSON string and still rendered as before;
//   - private: an object key in the private bucket plus what the admin needs to show it
//     (name, size, type). The key is never sent to a browser; staff get a short-lived
//     signed link through GET /inquiries/{id}/attachments/{index}.
type Attachment struct {
	URL  string `json:"url,omitempty"`
	Key  string `json:"key,omitempty"`
	Name string `json:"name,omitempty"`
	Size int64  `json:"size,omitempty"`
	Type string `json:"type,omitempty"`
}

// IsPrivate reports whether the file sits in the private bucket.
func (a Attachment) IsPrivate() bool { return a.Key != "" }

// MarshalJSON keeps a legacy attachment exactly as it was stored, a bare string, so rows
// written before private files existed are unchanged by a read-modify-write.
func (a Attachment) MarshalJSON() ([]byte, error) {
	if a.Key == "" && a.URL != "" && a.Name == "" && a.Size == 0 && a.Type == "" {
		return json.Marshal(a.URL)
	}
	type plain Attachment
	return json.Marshal(plain(a))
}

// UnmarshalJSON accepts the legacy string form and the object form.
func (a *Attachment) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var u string
		if err := json.Unmarshal(data, &u); err != nil {
			return err
		}
		*a = Attachment{URL: u}
		return nil
	}
	type plain Attachment
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*a = Attachment(p)
	return nil
}

const (
	// MaxAttachmentBytes is the cap per file. A site drawing or a phone photo fits; a
	// whole project folder does not, and that visitor is better served by Zalo.
	MaxAttachmentBytes = 15 << 20
	// MaxAttachmentsTotalBytes caps everything one lead carries.
	MaxAttachmentsTotalBytes = 30 << 20
	MaxAttachments           = 6

	maxAttachmentNameRunes = 120
)

// Content types stored with a private file, chosen from what the bytes are, never from
// what the browser claimed.
const (
	TypePDF  = "application/pdf"
	TypeDWG  = "image/vnd.dwg"
	TypeDXF  = "image/vnd.dxf"
	TypeJPEG = "image/jpeg"
	TypePNG  = "image/png"
	TypeWebP = "image/webp"
)

var (
	dwgMagic = regexp.MustCompile(`^AC10[0-9]{2}`)
	dxfBody  = regexp.MustCompile(`(?s)^[ \t]*(?:0|999)[ \t]*\r?\n`)
)

// DetectFileType identifies a file from its first bytes (pass at least the first 2 KB when
// the file is that long). It returns the content type and the extension to store under,
// or ok=false for anything that is not a PDF, DWG, DXF or JPEG/PNG/WebP image. ZIP, RAR,
// executables and documents with a renamed extension all fall in the ok=false case.
func DetectFileType(head []byte) (contentType, ext string, ok bool) {
	switch {
	case bytes.HasPrefix(head, []byte("%PDF-")):
		return TypePDF, "pdf", true
	case bytes.HasPrefix(head, []byte{0xFF, 0xD8, 0xFF}):
		return TypeJPEG, "jpg", true
	case bytes.HasPrefix(head, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return TypePNG, "png", true
	case len(head) >= 12 && bytes.HasPrefix(head, []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")):
		return TypeWebP, "webp", true
	case dwgMagic.Match(head):
		return TypeDWG, "dwg", true
	case isDXF(head):
		return TypeDXF, "dxf", true
	}
	return "", "", false
}

// isDXF recognises both DXF flavours: binary ("AutoCAD Binary DXF") and ASCII, which is a
// text file of "group code / value" lines that starts at group 0 (or a 999 comment) and
// holds a SECTION. An ASCII file must be free of NUL bytes, which keeps an arbitrary
// binary from passing just because it contains the word SECTION.
func isDXF(head []byte) bool {
	if bytes.HasPrefix(head, []byte("AutoCAD Binary DXF")) {
		return true
	}
	head = bytes.TrimPrefix(head, []byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM
	if bytes.IndexByte(head, 0) >= 0 || !dxfBody.Match(head) {
		return false
	}
	return bytes.Contains(head, []byte("SECTION"))
}

var attachmentKeyPattern = regexp.MustCompile(`^inquiries/[0-9]{6}/[0-9a-f]{32}\.(pdf|jpg|png|webp|dwg|dxf)$`)

// ValidAttachmentKey accepts only keys the upload endpoint itself produces, so a visitor
// cannot point a lead at some other object in the bucket.
func ValidAttachmentKey(key string) bool { return attachmentKeyPattern.MatchString(key) }

func validAttachmentType(t string) bool {
	switch t {
	case TypePDF, TypeDWG, TypeDXF, TypeJPEG, TypePNG, TypeWebP:
		return true
	}
	return false
}

// SanitizeAttachmentName makes a client-supplied file name safe to store and to show:
// directories dropped, control and bidirectional-override characters removed, length
// capped. The result is display text only; it is never used as part of a storage key.
func SanitizeAttachmentName(name string) string {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case unicode.IsControl(r), r == '‮', r == '‭', r == '‎', r == '‏', r == utf8.RuneError:
			continue
		}
		b.WriteRune(r)
	}
	out := strings.TrimSpace(b.String())
	if utf8.RuneCountInString(out) > maxAttachmentNameRunes {
		out = string([]rune(out)[:maxAttachmentNameRunes])
	}
	return out
}

// validateAttachments bounds what one lead may carry and normalises the private entries.
// It trusts nothing: the handler has already replaced Size and Type with what the store
// reports, and this still re-checks them.
func validateAttachments(attachments []Attachment) ([]Attachment, []string) {
	if len(attachments) > MaxAttachments {
		return nil, []string{"at most 6 attachments allowed"}
	}
	var total int64
	cleaned := make([]Attachment, 0, len(attachments))
	for _, a := range attachments {
		switch {
		case a.Key != "":
			if !ValidAttachmentKey(a.Key) {
				return nil, []string{"attachment key is not valid"}
			}
			if !validAttachmentType(a.Type) {
				return nil, []string{"attachment type is not allowed"}
			}
			if a.Size <= 0 || a.Size > MaxAttachmentBytes {
				return nil, []string{"each file must be 15 MB or smaller"}
			}
			total += a.Size
			cleaned = append(cleaned, Attachment{Key: a.Key, Name: SanitizeAttachmentName(a.Name), Size: a.Size, Type: a.Type})
		default:
			u := strings.TrimSpace(a.URL)
			if u == "" {
				return nil, []string{"attachment must not be empty"}
			}
			// Legacy public photos are rendered as images by the admin, so only plain https
			// URLs are accepted: no javascript:, data:, http: or relative values.
			parsed, err := url.Parse(u)
			if err != nil || parsed.Scheme != "https" || parsed.Host == "" || len(u) > 2048 {
				return nil, []string{"attachment must be an https url"}
			}
			cleaned = append(cleaned, Attachment{URL: u})
		}
	}
	if total > MaxAttachmentsTotalBytes {
		return nil, []string{"files together must be 30 MB or smaller"}
	}
	return cleaned, nil
}
