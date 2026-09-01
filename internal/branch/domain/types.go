package domain

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/media"
	"github.com/trvux/elc-go/internal/platform/seo"
)

// ImageAsset re-exports the shared media type — see product/domain/types.go's
// identical alias for why this is centralized rather than duplicated.
type ImageAsset = media.ImageAsset

var slugRegex = regexp.MustCompile("^[a-z0-9-]+$")

type Branch struct {
	id              string
	name            string
	slug            string
	address         string
	phone           string
	email           string
	mapsURL         string
	mapsEmbed       string
	provinceCode    *string
	provinceName    *string
	wardCode        *string
	wardName        *string
	postalCode      *string
	description     json.RawMessage
	images          []ImageAsset
	isPublished     bool
	orderIndex      int
	metaTitle       *string
	metaDescription *string
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

// NewBranch validates and creates a new branch entity from user input.
// provinceCode/provinceName/wardCode/wardName/postalCode are optional — they
// reference internal/shippingzone's provinces/wards reference data (by code,
// no DB foreign key across module boundaries) with the human-readable name
// denormalized alongside so shared/lib/seo-schema.ts (frontend) never needs
// a runtime lookup to build a real PostalAddress.
func NewBranch(
	name, slug, address, phone, email, mapsURL, mapsEmbed string,
	provinceCode, provinceName, wardCode, wardName, postalCode *string,
	description json.RawMessage,
	images []ImageAsset,
	isPublished bool,
	orderIndex int,
	metaTitle, metaDescription *string,
) (*Branch, error) {
	fields := map[string][]string{}

	if errs := validateName(name); len(errs) > 0 {
		fields["name"] = errs
	}
	if errs := validateSlug(slug); len(errs) > 0 {
		fields["slug"] = errs
	}
	if errs := validateAddress(address); len(errs) > 0 {
		fields["address"] = errs
	}
	if errs := validatePhone(phone); len(errs) > 0 {
		fields["phone"] = errs
	}
	if errs := validateEmail(email); len(errs) > 0 {
		fields["email"] = errs
	}
	if errs := validateMapsURL(mapsURL); len(errs) > 0 {
		fields["mapsUrl"] = errs
	}
	if errs := validateMapsEmbed(mapsEmbed); len(errs) > 0 {
		fields["mapsEmbed"] = errs
	}
	if errs := seo.ValidateMetaTitle(metaTitle); len(errs) > 0 {
		fields["metaTitle"] = errs
	}
	if errs := seo.ValidateMetaDescription(metaDescription); len(errs) > 0 {
		fields["metaDescription"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	now := time.Now()
	return &Branch{
		name:            name,
		slug:            slug,
		address:         address,
		phone:           phone,
		email:           email,
		mapsURL:         mapsURL,
		mapsEmbed:       mapsEmbed,
		provinceCode:    provinceCode,
		provinceName:    provinceName,
		wardCode:        wardCode,
		wardName:        wardName,
		postalCode:      postalCode,
		description:     description,
		images:          images,
		isPublished:     isPublished,
		orderIndex:      orderIndex,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// RehydrateBranch reconstructs a branch entity from a trusted DB row.
func RehydrateBranch(
	id, name, slug, address, phone, email, mapsURL, mapsEmbed string,
	provinceCode, provinceName, wardCode, wardName, postalCode *string,
	description json.RawMessage,
	images []ImageAsset,
	isPublished bool,
	orderIndex int,
	metaTitle, metaDescription *string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *Branch {
	return &Branch{
		id:              id,
		name:            name,
		slug:            slug,
		address:         address,
		phone:           phone,
		email:           email,
		mapsURL:         mapsURL,
		mapsEmbed:       mapsEmbed,
		provinceCode:    provinceCode,
		provinceName:    provinceName,
		wardCode:        wardCode,
		wardName:        wardName,
		postalCode:      postalCode,
		description:     description,
		images:          images,
		isPublished:     isPublished,
		orderIndex:      orderIndex,
		metaTitle:       metaTitle,
		metaDescription: metaDescription,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		deletedAt:       deletedAt,
	}
}

func (b *Branch) ID() string                   { return b.id }
func (b *Branch) Name() string                 { return b.name }
func (b *Branch) Slug() string                 { return b.slug }
func (b *Branch) Address() string              { return b.address }
func (b *Branch) Phone() string                { return b.phone }
func (b *Branch) Email() string                { return b.email }
func (b *Branch) MapsURL() string              { return b.mapsURL }
func (b *Branch) MapsEmbed() string            { return b.mapsEmbed }
func (b *Branch) ProvinceCode() *string        { return b.provinceCode }
func (b *Branch) ProvinceName() *string        { return b.provinceName }
func (b *Branch) WardCode() *string            { return b.wardCode }
func (b *Branch) WardName() *string            { return b.wardName }
func (b *Branch) PostalCode() *string          { return b.postalCode }
func (b *Branch) Description() json.RawMessage { return b.description }
func (b *Branch) Images() []ImageAsset         { return b.images }
func (b *Branch) IsPublished() bool            { return b.isPublished }
func (b *Branch) OrderIndex() int              { return b.orderIndex }
func (b *Branch) MetaTitle() *string           { return b.metaTitle }
func (b *Branch) MetaDescription() *string     { return b.metaDescription }
func (b *Branch) CreatedAt() time.Time         { return b.createdAt }
func (b *Branch) UpdatedAt() time.Time         { return b.updatedAt }
func (b *Branch) DeletedAt() *time.Time        { return b.deletedAt }

func (b *Branch) IsDeleted() bool {
	return b.deletedAt != nil
}

func (b *Branch) Reorder(orderIndex int) {
	b.orderIndex = orderIndex
	b.updatedAt = time.Now()
}

// Update applies a partial edit from a form submission: only non-nil fields
// in input are validated and set. Validates and applies in the same order
// the fields appear below, failing fast on the first invalid field (no error
// aggregation) — this matches the single call site in
// application.UpdateBranch, which always submits the whole edit form at once.
//
// OrderIndex goes through Reorder rather than setting orderIndex directly,
// so drag-drop reorder (application.UpdateBranchOrder) and the edit form
// share one path for that field.
func (b *Branch) Update(input UpdateBranchInput) error {
	// changed tracks whether any field below actually mutated b, so a no-op
	// call (all fields nil, or metaTitle/metaDescription unchanged) leaves
	// updatedAt untouched — matching the old per-field UpdateX() behavior,
	// which only ever bumped updatedAt from inside a branch that ran.
	// OrderIndex is excluded: Reorder bumps updatedAt itself.
	changed := false

	if input.Name != nil {
		if errs := validateName(*input.Name); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"name": errs})
		}
		b.name = *input.Name
		changed = true
	}
	if input.Slug != nil {
		if errs := validateSlug(*input.Slug); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
		}
		b.slug = *input.Slug
		changed = true
	}
	if input.Address != nil {
		if errs := validateAddress(*input.Address); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"address": errs})
		}
		b.address = *input.Address
		changed = true
	}
	if input.Phone != nil {
		if errs := validatePhone(*input.Phone); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"phone": errs})
		}
		b.phone = *input.Phone
		changed = true
	}
	if input.Email != nil {
		if errs := validateEmail(*input.Email); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"email": errs})
		}
		b.email = *input.Email
		changed = true
	}
	if input.MapsURL != nil {
		if errs := validateMapsURL(*input.MapsURL); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"mapsUrl": errs})
		}
		b.mapsURL = *input.MapsURL
		changed = true
	}
	if input.MapsEmbed != nil {
		if errs := validateMapsEmbed(*input.MapsEmbed); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"mapsEmbed": errs})
		}
		b.mapsEmbed = *input.MapsEmbed
		changed = true
	}
	// Bundled: the frontend's cascading combobox always submits
	// province/ward code+name+postalCode together (see BranchManagement.tsx),
	// so treat "any one present" as "replace the whole location" rather than
	// merging field-by-field.
	if input.ProvinceCode != nil || input.ProvinceName != nil || input.WardCode != nil || input.WardName != nil || input.PostalCode != nil {
		b.provinceCode = input.ProvinceCode
		b.provinceName = input.ProvinceName
		b.wardCode = input.WardCode
		b.wardName = input.WardName
		b.postalCode = input.PostalCode
		changed = true
	}
	if input.Description != nil {
		b.description = input.Description
		changed = true
	}
	if input.Images != nil {
		b.images = input.Images
		changed = true
	}
	if input.IsPublished != nil {
		b.isPublished = *input.IsPublished
		changed = true
	}
	if input.OrderIndex != nil {
		b.Reorder(*input.OrderIndex)
	}
	if input.MetaTitle != nil && !seo.Unchanged(b.metaTitle, input.MetaTitle) {
		if errs := seo.ValidateMetaTitle(input.MetaTitle); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"metaTitle": errs})
		}
		b.metaTitle = input.MetaTitle
		changed = true
	}
	if input.MetaDescription != nil && !seo.Unchanged(b.metaDescription, input.MetaDescription) {
		if errs := seo.ValidateMetaDescription(input.MetaDescription); len(errs) > 0 {
			return apperr.NewValidationError("validation failed", map[string][]string{"metaDescription": errs})
		}
		b.metaDescription = input.MetaDescription
		changed = true
	}
	if changed {
		b.updatedAt = time.Now()
	}
	return nil
}

func (b *Branch) MarkDeleted(deletedAt time.Time) {
	b.deletedAt = &deletedAt
}

func (b *Branch) Restore() {
	b.deletedAt = nil
	b.updatedAt = time.Now()
}

func validateName(name string) []string {
	if name == "" {
		return []string{"Tên chi nhánh không được để trống"}
	}
	if utf8.RuneCountInString(name) > 100 {
		return []string{"Tên chi nhánh không được quá 100 ký tự"}
	}
	return nil
}

func validateSlug(slug string) []string {
	if slug == "" {
		return []string{"Slug không được để trống"}
	}
	if utf8.RuneCountInString(slug) > 100 {
		return []string{"Slug không được quá 100 ký tự"}
	}
	if !slugRegex.MatchString(slug) {
		return []string{"Slug chỉ được chứa chữ thường, số và dấu gạch ngang"}
	}
	return nil
}

func validateAddress(address string) []string {
	if address == "" {
		return []string{"Địa chỉ không được để trống"}
	}
	return nil
}

func validatePhone(phone string) []string {
	if phone == "" {
		return []string{"Số điện thoại không được để trống"}
	}
	return nil
}

func validateEmail(email string) []string {
	if email == "" {
		return []string{"Email không hợp lệ"}
	}
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return []string{"Email không hợp lệ"}
	}
	return nil
}

func validateMapsURL(mapsURL string) []string {
	if mapsURL == "" {
		return []string{"URL bản đồ không hợp lệ"}
	}
	_, err := url.ParseRequestURI(mapsURL)
	if err != nil {
		return []string{"URL bản đồ không hợp lệ"}
	}
	return nil
}

// mapsEmbedTagRe matches the ENTIRE trimmed input against exactly one bare
// <iframe ...></iframe> (or self-closed) tag — attributes may not contain
// '<'/'>' themselves, which rules out smuggling a second tag (e.g. <script>)
// inside what looks like an attribute value. Google's own "Share > Embed a
// map" snippet is always this shape; this field should never hold anything
// richer (surrounding text, multiple tags).
var mapsEmbedTagRe = regexp.MustCompile(`(?is)^<iframe\s+([^<>]*?)\s*/?>(?:\s*</iframe>)?$`)

// mapsEmbedAttrRe pulls out one name="value" (or bare name) pair at a time
// from the tag's attribute string.
var mapsEmbedAttrRe = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9-]*)(?:\s*=\s*"([^"]*)")?`)

// mapsEmbedAllowedAttrs is every attribute Google's own embed snippet
// actually emits — a whitelist, not a blacklist, so an attribute this list
// doesn't recognize (onload, onerror, style with a javascript: url, ...) is
// rejected by default instead of needing its own explicit block rule.
var mapsEmbedAllowedAttrs = map[string]bool{
	"src": true, "width": true, "height": true,
	"frameborder": true, "allowfullscreen": true, "loading": true,
	"referrerpolicy": true, "title": true, "style": true,
}

// mapsEmbedAllowedSrcHosts restricts src to Google's own embed domain — this
// field is only ever meant to carry a Google Maps iframe, never arbitrary
// third-party markup.
var mapsEmbedAllowedSrcHosts = map[string]bool{
	"www.google.com": true, "google.com": true, "maps.google.com": true,
}

// mapsEmbedStyleRe is the ONLY style value Google's own embed snippet ever
// emits ("border:0;" or "border:0"). Whitelisting the attribute NAME alone
// isn't enough for `style` — its VALUE is free-form CSS and was still an
// injection vector even with src/other attributes locked down (e.g.
// background:url(...) pointing at an attacker's domain for tracking/
// exfiltration, or position:fixed to overlay/deface the page) — caught in
// code review on the first version of this fix. Every other allowed
// attribute's value is already independently constrained (src by
// mapsEmbedAllowedSrcHosts; width/height/frameborder/allowfullscreen/
// loading/referrerpolicy/title are inert w.r.t. injection), so only style
// needs its own value-level allowlist.
var mapsEmbedStyleRe = regexp.MustCompile(`^border:\s*0;?$`)

// validateMapsEmbed is a best-effort backend guard against stored XSS via
// this field (a raw HTML snippet an admin pastes from Google Maps): it
// rejects anything that isn't a single <iframe> tag with a whitelisted
// attribute set and a src pointing at Google's own domain, rather than
// attempting to sanitize arbitrary HTML. See docs/rfc/2026-09-02-backend-
// code-review-round2.md §3.9 for why this exists — whether the frontend
// renders this field as raw HTML at all wasn't confirmed at review time.
func validateMapsEmbed(mapsEmbed string) []string {
	trimmed := strings.TrimSpace(mapsEmbed)
	if trimmed == "" {
		return []string{"Mã nhúng bản đồ không được để trống"}
	}

	m := mapsEmbedTagRe.FindStringSubmatch(trimmed)
	if m == nil {
		return []string{"Mã nhúng bản đồ phải là đúng 1 thẻ <iframe> lấy từ Google Maps (Chia sẻ > Nhúng bản đồ), không chứa nội dung nào khác"}
	}

	hasSrc := false
	for _, attr := range mapsEmbedAttrRe.FindAllStringSubmatch(m[1], -1) {
		name, value := strings.ToLower(attr[1]), attr[2]
		if !mapsEmbedAllowedAttrs[name] {
			return []string{fmt.Sprintf("Mã nhúng bản đồ chứa thuộc tính không được phép: %s", name)}
		}
		if name == "src" {
			hasSrc = true
			u, err := url.Parse(value)
			if err != nil || u.Scheme != "https" || !mapsEmbedAllowedSrcHosts[u.Hostname()] {
				return []string{"Mã nhúng bản đồ phải trỏ tới google.com (Google Maps)"}
			}
		}
		if name == "style" && !mapsEmbedStyleRe.MatchString(strings.TrimSpace(value)) {
			return []string{"Mã nhúng bản đồ chứa giá trị style không được phép"}
		}
	}
	if !hasSrc {
		return []string{"Mã nhúng bản đồ phải có thuộc tính src trỏ tới Google Maps"}
	}
	return nil
}

type CreateBranchInput struct {
	Name            string
	Slug            string
	Address         string
	Phone           string
	Email           string
	MapsURL         string
	MapsEmbed       string
	ProvinceCode    *string
	ProvinceName    *string
	WardCode        *string
	WardName        *string
	PostalCode      *string
	Description     json.RawMessage
	Images          []ImageAsset
	IsPublished     bool
	OrderIndex      int
	MetaTitle       *string
	MetaDescription *string
}

type UpdateBranchInput struct {
	ID              string
	Name            *string
	Slug            *string
	Address         *string
	Phone           *string
	Email           *string
	MapsURL         *string
	MapsEmbed       *string
	ProvinceCode    *string
	ProvinceName    *string
	WardCode        *string
	WardName        *string
	PostalCode      *string
	Description     json.RawMessage
	Images          []ImageAsset
	IsPublished     *bool
	OrderIndex      *int
	MetaTitle       *string
	MetaDescription *string
}

type BranchFilter struct {
	IsPublished *bool
	Search      string
	Limit       int
	Offset      int
}
