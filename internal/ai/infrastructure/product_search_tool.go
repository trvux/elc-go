package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/trvux/elc-go/internal/ai/domain"
	productdomain "github.com/trvux/elc-go/internal/product/domain"
)

// searchProductsArgs is the JSON the model sends when it calls
// search_products — the tool's Parameters schema below documents these same
// fields to the model.
type searchProductsArgs struct {
	Query        string `json:"query"`
	CategorySlug string `json:"category_slug"`
	MinPrice     *int64 `json:"min_price"`
	MaxPrice     *int64 `json:"max_price"`
}

type productSummary struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Brand       string `json:"brand,omitempty"`
	Category    string `json:"category,omitempty"`
	PriceFrom   *int64 `json:"price_from,omitempty"`
	PriceTo     *int64 `json:"price_to,omitempty"`
	StockStatus string `json:"stock_status,omitempty"`
}

// searchResultLimit caps how many products the tool returns per call — keeps
// the tool-result message small and matches what a sales rep would actually
// list in one reply.
const searchResultLimit = 5

// NewProductSearchTool builds the search_products tool definition plus its
// executor, backed by repo — the same ProductRepository the product module's
// own HTTP handlers use, so the assistant's answers stay grounded in the
// real catalog (price, stock, publish status) instead of the model
// inventing a product or price.
func NewProductSearchTool(repo productdomain.ProductRepository) (domain.ToolDefinition, func(ctx context.Context, argumentsJSON string) (string, error)) {
	def := domain.ToolDefinition{
		Name: "search_products",
		Description: "Search the store's published product catalog by keyword, category slug, " +
			"and/or price range (VND). Always call this before recommending or quoting a price for " +
			"any specific product — never invent a product name or price.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {"type": "string", "description": "Free-text keyword, e.g. a product name or type"},
				"category_slug": {"type": "string", "description": "Known category slug, e.g. may-lanh-treo-tuong"},
				"min_price": {"type": "integer", "description": "Minimum price in VND"},
				"max_price": {"type": "integer", "description": "Maximum price in VND"}
			}
		}`),
	}

	execute := func(ctx context.Context, argumentsJSON string) (string, error) {
		var args searchProductsArgs
		if argumentsJSON != "" {
			if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
				return "", fmt.Errorf("ai: invalid search_products arguments: %w", err)
			}
		}

		status := productdomain.ProductStatusPublished
		filter := productdomain.ProductFilter{
			Status:   &status,
			Search:   args.Query,
			MinPrice: args.MinPrice,
			MaxPrice: args.MaxPrice,
			Limit:    searchResultLimit,
			SortBy:   productdomain.SortByPriceAsc,
		}
		if args.CategorySlug != "" {
			filter.CategorySlugs = []string{args.CategorySlug}
		}

		result, err := repo.GetAll(ctx, filter)
		if err != nil {
			return "", fmt.Errorf("ai: search_products query: %w", err)
		}

		summaries := make([]productSummary, 0, len(result.Products))
		for _, p := range result.Products {
			s := productSummary{Name: p.Name(), Slug: p.Slug(), PriceFrom: p.PriceMin(), PriceTo: p.PriceMax()}
			if p.Brand != nil {
				s.Brand = p.Brand.Name
			}
			if p.Category != nil {
				s.Category = p.Category.Name
			}
			if p.DisplayStockStatus() != nil {
				s.StockStatus = *p.DisplayStockStatus()
			}
			summaries = append(summaries, s)
		}

		out, err := json.Marshal(summaries)
		if err != nil {
			return "", fmt.Errorf("ai: encode search_products result: %w", err)
		}
		return string(out), nil
	}

	return def, execute
}
