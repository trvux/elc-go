package domain

import (
	"encoding/json"
	"time"
)

// CatalogPage is the singleton content/SEO config for "Tất cả sản phẩm" (the
// catch-all/root product listing) — it belongs to Product's own bounded
// context (not the generic system_pages module, which serves an unrelated
// set of pages: homepage/news-hub/policy). There is exactly one row, never
// created/deleted, only read and updated — see
// infrastructure/catalog_page_repository.go.
type CatalogPage struct {
	content         json.RawMessage
	metaTitle       *string
	metaDescription *string
	updatedAt       time.Time
}

// RehydrateCatalogPage reconstructs from the single trusted DB row — no
// validation, only the infrastructure layer should call this.
func RehydrateCatalogPage(content json.RawMessage, metaTitle, metaDescription *string, updatedAt time.Time) *CatalogPage {
	return &CatalogPage{content: content, metaTitle: metaTitle, metaDescription: metaDescription, updatedAt: updatedAt}
}

func (p *CatalogPage) Content() json.RawMessage { return p.content }
func (p *CatalogPage) MetaTitle() *string       { return p.metaTitle }
func (p *CatalogPage) MetaDescription() *string { return p.metaDescription }
func (p *CatalogPage) UpdatedAt() time.Time     { return p.updatedAt }

type UpdateCatalogPageInput struct {
	Content         json.RawMessage
	MetaTitle       *string
	MetaDescription *string
}
