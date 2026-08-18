package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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

// specEntry is one rendered spec, e.g. {"label": "Công suất làm lạnh",
// "value": "9000 BTU/h"} — see renderAttributeValue for how each
// AttributeValueRef data_type becomes a display string.
type specEntry struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type productSummary struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Brand       string `json:"brand,omitempty"`
	Category    string `json:"category,omitempty"`
	PriceFrom   *int64 `json:"price_from,omitempty"`
	PriceTo     *int64 `json:"price_to,omitempty"`
	StockStatus string `json:"stock_status,omitempty"`
	// Highlights/Specs ground the model in the product's real recorded
	// features/technical specs — without these, a question about a
	// specific feature or spec has nothing here to answer from, and the
	// model falls back to its general training knowledge instead of this
	// store's actual catalog (the exact failure mode this fixes).
	Highlights []string    `json:"highlights,omitempty"`
	Specs      []specEntry `json:"specs,omitempty"`
}

// searchResultLimit caps how many products the tool returns per call — keeps
// the tool-result message small and matches what a sales rep would actually
// list in one reply.
const searchResultLimit = 5

// NewProductSearchTool builds the search_products tool definition plus its
// executor, backed by repo — the same ProductRepository the product module's
// own HTTP handlers use, so the assistant's answers stay grounded in the
// real catalog (price, stock, publish status, specs) instead of the model
// inventing a product, price, or feature.
func NewProductSearchTool(repo productdomain.ProductRepository) (domain.ToolDefinition, func(ctx context.Context, argumentsJSON string) (string, error)) {
	def := domain.ToolDefinition{
		Name: "search_products",
		Description: "Search the store's published product catalog by keyword, category slug, " +
			"and/or price range (VND). Returns each match's price, stock, highlights, and technical " +
			"specs. Always call this before recommending or quoting a price, feature, or spec for " +
			"any specific product — never invent a product name, price, or spec.",
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

		// GetAll (a list query) never populates AttributeValues — see
		// ProductRepository's doc comment — so a second, bounded (≤
		// searchResultLimit) lookup is needed to ground specs too. Reuses
		// GetByIDsWithAttributeValues as-is (already built for Compare
		// Products), no new SQL.
		specsByID, err := fetchSpecsByID(ctx, repo, result.Products)
		if err != nil {
			return "", fmt.Errorf("ai: search_products specs query: %w", err)
		}

		summaries := make([]productSummary, 0, len(result.Products))
		for _, p := range result.Products {
			s := productSummary{
				Name: p.Name(), Slug: p.Slug(), PriceFrom: p.PriceMin(), PriceTo: p.PriceMax(),
				Highlights: p.Highlights(), Specs: renderSpecs(specsByID[p.ID()]),
			}
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

func fetchSpecsByID(ctx context.Context, repo productdomain.ProductRepository, products []*productdomain.ProductWithRelations) (map[string][]productdomain.AttributeValueRef, error) {
	if len(products) == 0 {
		return nil, nil
	}
	ids := make([]string, len(products))
	for i, p := range products {
		ids[i] = p.ID()
	}

	withSpecs, err := repo.GetByIDsWithAttributeValues(ctx, ids)
	if err != nil {
		return nil, err
	}

	byID := make(map[string][]productdomain.AttributeValueRef, len(withSpecs))
	for _, p := range withSpecs {
		byID[p.ID()] = p.AttributeValues
	}
	return byID, nil
}

// renderSpecs turns each AttributeValueRef into a label/value pair the
// model can read directly — skips any attribute with no value recorded at
// all, so the tool result doesn't pad the prompt with empty specs.
func renderSpecs(values []productdomain.AttributeValueRef) []specEntry {
	if len(values) == 0 {
		return nil
	}
	specs := make([]specEntry, 0, len(values))
	for _, v := range values {
		value := renderAttributeValue(v)
		if value == "" {
			continue
		}
		specs = append(specs, specEntry{Label: v.Name, Value: value})
	}
	if len(specs) == 0 {
		return nil
	}
	return specs
}

// renderAttributeValue picks the one populated value field for v's
// data_type — see AttributeValueRef's doc comment: exactly one of
// ValueText/ValueNumber/ValueBoolean/ValueOptions is meaningful per row.
func renderAttributeValue(v productdomain.AttributeValueRef) string {
	switch {
	case v.ValueText != nil && *v.ValueText != "":
		return *v.ValueText
	case v.ValueNumber != nil:
		s := strconv.FormatFloat(*v.ValueNumber, 'f', -1, 64)
		if v.Unit != nil && *v.Unit != "" {
			return s + " " + *v.Unit
		}
		return s
	case v.ValueBoolean != nil:
		if *v.ValueBoolean {
			return "Có"
		}
		return "Không"
	case len(v.ValueOptions) > 0:
		return strings.Join(v.ValueOptions, ", ")
	default:
		return ""
	}
}
