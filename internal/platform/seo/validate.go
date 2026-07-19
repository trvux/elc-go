// Package seo holds the meta title/description length limits shared by
// every content module (news, product, project) that carries its own SEO
// fields — before this, only news validated at all (≤70/≤160), so a
// product/project title written past what Google actually displays would
// silently get truncated with "..." instead of being caught at write time.
package seo

import "unicode/utf8"

const (
	MaxMetaTitleLength       = 60
	MaxMetaDescriptionLength = 160
)

func ValidateMetaTitle(metaTitle *string) []string {
	if metaTitle != nil && utf8.RuneCountInString(*metaTitle) > MaxMetaTitleLength {
		return []string{"Tiêu đề SEO không nên quá 60 ký tự"}
	}
	return nil
}

func ValidateMetaDescription(metaDescription *string) []string {
	if metaDescription != nil && utf8.RuneCountInString(*metaDescription) > MaxMetaDescriptionLength {
		return []string{"Mô tả SEO không nên quá 160 ký tự"}
	}
	return nil
}
