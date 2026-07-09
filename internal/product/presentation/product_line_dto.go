package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/product/domain"
)

type productLineResponse struct {
	ID          string     `json:"id"`
	BrandID     string     `json:"brand_id"`
	CategoryID  *string    `json:"category_id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	TierRank    int        `json:"tier_rank"`
	Description *string    `json:"description"`
	MpnPrefixes []string   `json:"mpn_prefixes"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

func toProductLineResponse(l *domain.ProductLine) productLineResponse {
	return productLineResponse{
		ID: l.ID(), BrandID: l.BrandID(), CategoryID: l.CategoryID(), Code: l.Code(), Name: l.Name(),
		TierRank: l.TierRank(), Description: l.Description(), MpnPrefixes: l.MpnPrefixes(),
		CreatedAt: l.CreatedAt(), UpdatedAt: l.UpdatedAt(), DeletedAt: l.DeletedAt(),
	}
}

func toProductLineResponseList(lines []*domain.ProductLine) []productLineResponse {
	result := make([]productLineResponse, len(lines))
	for i, l := range lines {
		result[i] = toProductLineResponse(l)
	}
	return result
}

type createProductLineRequest struct {
	BrandID     string   `json:"brand_id"`
	CategoryID  *string  `json:"category_id,omitempty"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	TierRank    int      `json:"tier_rank"`
	Description *string  `json:"description,omitempty"`
	MpnPrefixes []string `json:"mpn_prefixes,omitempty"`
}

type updateProductLineRequest struct {
	CategoryID  *string  `json:"category_id"`
	Name        string   `json:"name"`
	TierRank    int      `json:"tier_rank"`
	Description *string  `json:"description"`
	MpnPrefixes []string `json:"mpn_prefixes"`
}
