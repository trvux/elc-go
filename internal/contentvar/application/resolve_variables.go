package application

import (
	"context"
	"fmt"

	branddomain "github.com/trvux/elc-go/internal/brand/domain"
	categorydomain "github.com/trvux/elc-go/internal/category/domain"
	"github.com/trvux/elc-go/internal/contentvar/domain"
	groupdomain "github.com/trvux/elc-go/internal/group/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
	productdomain "github.com/trvux/elc-go/internal/product/domain"
)

// Resolver batch-resolves content variables against the live catalog.
// VariableFilter is expressed in slugs (the stable identifiers a content
// editor UI can safely let an author pick), so every request is first
// translated into the CategorySlugs/BrandID shape productdomain.ProductFilter
// already uses for every other product listing/facet query.
type Resolver struct {
	products   productdomain.ProductRepository
	categories categorydomain.CategoryRepository
	groups     groupdomain.GroupRepository
	brands     branddomain.BrandRepository
}

func NewResolver(
	products productdomain.ProductRepository,
	categories categorydomain.CategoryRepository,
	groups groupdomain.GroupRepository,
	brands branddomain.BrandRepository,
) *Resolver {
	return &Resolver{products: products, categories: categories, groups: groups, brands: brands}
}

// filterScope caches the count/price lookups for one resolved product
// filter across every request that shares it — e.g. a "count" and a
// "priceMin" for the same category only hit the database once each,
// instead of once per requested variable.
type filterScope struct {
	filter productdomain.ProductFilter
	// categorySlugs is the resolved scope's category slugs — kept
	// alongside filter so MetricCategoryCount can read its length without
	// a products query (see resolveMetric).
	categorySlugs []string
	count         *int
	price         *productdomain.PriceFacet
	brandCount    *int
}

// Resolve returns one value per request, keyed by VariableRequest.ID — a
// request that fails to resolve (unconfigured scope on a freshly-inserted
// node, a slug that no longer exists) is simply omitted from the result
// rather than failing the whole batch: this runs on every public /san-pham
// render, and one bad content variable on a page must never blank out
// every OTHER number on that same page.
func (s *Resolver) Resolve(ctx context.Context, requests []domain.VariableRequest) (map[string]any, error) {
	values := make(map[string]any, len(requests))
	groupCategorySlugs := map[string][]string{}
	brandIDs := map[string]string{}
	scopes := map[string]*filterScope{}

	for _, req := range requests {
		categorySlugs, err := s.resolveCategorySlugs(ctx, req.Filter, groupCategorySlugs)
		if err != nil {
			continue
		}
		brandID, err := s.resolveBrandID(ctx, req.Filter.BrandSlug, brandIDs)
		if err != nil {
			continue
		}

		scopeKey := fmt.Sprintf("%v|%s|%s:%s", categorySlugs, brandID, req.Filter.AttributeCode, req.Filter.AttributeValue)
		scope, ok := scopes[scopeKey]
		if !ok {
			filter := productdomain.ProductFilter{CategorySlugs: categorySlugs}
			if brandID != "" {
				filter.BrandID = &brandID
			}
			if req.Filter.AttributeCode != "" && req.Filter.AttributeValue != "" {
				filter.AttributeTokens = []string{req.Filter.AttributeCode + ":" + req.Filter.AttributeValue}
			}
			scope = &filterScope{filter: filter, categorySlugs: categorySlugs}
			scopes[scopeKey] = scope
		}

		value, err := s.resolveMetric(ctx, scope, req.Metric)
		if err != nil {
			continue
		}
		values[req.ID] = value
	}

	return values, nil
}

func (s *Resolver) resolveMetric(ctx context.Context, scope *filterScope, metric domain.Metric) (any, error) {
	switch metric {
	case domain.MetricCount:
		if scope.count == nil {
			count, err := s.products.Count(ctx, scope.filter)
			if err != nil {
				return nil, fmt.Errorf("contentvar resolver count: %w", err)
			}
			scope.count = &count
		}
		return *scope.count, nil
	case domain.MetricPriceMin, domain.MetricPriceMax:
		if scope.price == nil {
			price, err := s.products.PriceRange(ctx, scope.filter)
			if err != nil {
				return nil, fmt.Errorf("contentvar resolver price range: %w", err)
			}
			scope.price = &price
		}
		if metric == domain.MetricPriceMin {
			return scope.price.Min, nil
		}
		return scope.price.Max, nil
	case domain.MetricBrandCount:
		if scope.brandCount == nil {
			count, err := s.products.BrandCount(ctx, scope.filter)
			if err != nil {
				return nil, fmt.Errorf("contentvar resolver brand count: %w", err)
			}
			scope.brandCount = &count
		}
		return *scope.brandCount, nil
	case domain.MetricCategoryCount:
		return len(scope.categorySlugs), nil
	default:
		return nil, apperr.NewValidationError("invalid metric", map[string][]string{
			"metric": {string(metric)},
		})
	}
}

func (s *Resolver) resolveCategorySlugs(ctx context.Context, filter domain.VariableFilter, cache map[string][]string) ([]string, error) {
	if filter.CategorySlug != "" {
		return []string{filter.CategorySlug}, nil
	}
	if filter.GroupSlug == "" {
		return nil, apperr.NewValidationError("invalid filter", map[string][]string{
			"filter": {"groupSlug or categorySlug is required"},
		})
	}
	if slugs, ok := cache[filter.GroupSlug]; ok {
		return slugs, nil
	}

	group, err := s.groups.GetBySlug(ctx, filter.GroupSlug)
	if err != nil {
		return nil, fmt.Errorf("contentvar resolver group %q: %w", filter.GroupSlug, err)
	}
	if group == nil {
		return nil, apperr.NewNotFoundError(fmt.Sprintf("group %q", filter.GroupSlug))
	}

	cats, err := s.categories.GetAll(ctx, categorydomain.CategoryFilter{GroupID: group.ID()})
	if err != nil {
		return nil, fmt.Errorf("contentvar resolver categories under group %q: %w", filter.GroupSlug, err)
	}
	slugs := make([]string, len(cats))
	for i, c := range cats {
		slugs[i] = c.Slug()
	}
	cache[filter.GroupSlug] = slugs
	return slugs, nil
}

func (s *Resolver) resolveBrandID(ctx context.Context, slug string, cache map[string]string) (string, error) {
	if slug == "" {
		return "", nil
	}
	if id, ok := cache[slug]; ok {
		return id, nil
	}

	brand, err := s.brands.GetBySlug(ctx, slug)
	if err != nil {
		return "", fmt.Errorf("contentvar resolver brand %q: %w", slug, err)
	}
	if brand == nil {
		return "", apperr.NewNotFoundError(fmt.Sprintf("brand %q", slug))
	}
	cache[slug] = brand.ID()
	return brand.ID(), nil
}
