package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/wishlist/domain"
)

type productSummaryResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	ImageURL     string `json:"image_url"`
	DisplayPrice *int64 `json:"display_price"`
}

type wishlistItemResponse struct {
	ID        string                  `json:"id"`
	ProductID string                  `json:"product_id"`
	CreatedAt time.Time               `json:"created_at"`
	Product   *productSummaryResponse `json:"product"`
}

func toWishlistItemResponse(item *domain.WishlistItemWithProduct) wishlistItemResponse {
	var product *productSummaryResponse
	if item.Product != nil {
		product = &productSummaryResponse{
			ID: item.Product.ID, Name: item.Product.Name, Slug: item.Product.Slug,
			ImageURL: item.Product.ImageURL, DisplayPrice: item.Product.DisplayPrice,
		}
	}
	return wishlistItemResponse{
		ID: item.ID(), ProductID: item.ProductID(), CreatedAt: item.CreatedAt(),
		Product: product,
	}
}

func toWishlistItemResponseList(items []*domain.WishlistItemWithProduct) []wishlistItemResponse {
	result := make([]wishlistItemResponse, len(items))
	for i, item := range items {
		result[i] = toWishlistItemResponse(item)
	}
	return result
}

type addWishlistItemRequest struct {
	ProductID string `json:"product_id"`
}
