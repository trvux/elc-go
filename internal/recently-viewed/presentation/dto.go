package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/recently-viewed/domain"
)

type productSummaryResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	ImageURL     string `json:"image_url"`
	DisplayPrice *int64 `json:"display_price"`
}

type recentlyViewedItemResponse struct {
	ID        string                  `json:"id"`
	ProductID string                  `json:"product_id"`
	ViewedAt  time.Time               `json:"viewed_at"`
	Product   *productSummaryResponse `json:"product"`
}

func toRecentlyViewedItemResponse(item *domain.RecentlyViewedItemWithProduct) recentlyViewedItemResponse {
	var product *productSummaryResponse
	if item.Product != nil {
		product = &productSummaryResponse{
			ID: item.Product.ID, Name: item.Product.Name, Slug: item.Product.Slug,
			ImageURL: item.Product.ImageURL, DisplayPrice: item.Product.DisplayPrice,
		}
	}
	return recentlyViewedItemResponse{
		ID: item.ID(), ProductID: item.ProductID(), ViewedAt: item.ViewedAt(),
		Product: product,
	}
}

func toRecentlyViewedItemResponseList(items []*domain.RecentlyViewedItemWithProduct) []recentlyViewedItemResponse {
	result := make([]recentlyViewedItemResponse, len(items))
	for i, item := range items {
		result[i] = toRecentlyViewedItemResponse(item)
	}
	return result
}

type recordRecentlyViewedRequest struct {
	ProductID string `json:"product_id"`
}
