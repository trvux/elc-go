// Package domain defines content variables — placeholders SEO content
// (Tiptap JSON docs) can embed instead of hardcoding a number that will
// drift once the catalog changes (e.g. "130 sản phẩm", "59 mẫu treo
// tường", a price range). The page render step resolves them against the
// live catalog right before serving HTML, so Google always crawls the
// current number.
package domain

// Metric identifies which aggregate a content variable resolves to.
type Metric string

const (
	MetricCount    Metric = "count"
	MetricPriceMin Metric = "priceMin"
	MetricPriceMax Metric = "priceMax"
	// MetricBrandCount is the number of DISTINCT brands with at least one
	// product in scope (e.g. "2 thương hiệu" for the máy lạnh group) —
	// computed from the products table, so a brand with zero products in
	// this scope never counts, unlike a plain lookup of the brands table.
	MetricBrandCount Metric = "brandCount"
	// MetricCategoryCount is the number of categories under a group scope
	// (e.g. "5 loại lắp đặt") — a catalog-structure fact (how many
	// installation types this group offers), not stock-dependent, so it's
	// just the resolved category slug count, no products query needed.
	// Meaningless (always 1) for a plain CategorySlug scope.
	MetricCategoryCount Metric = "categoryCount"
)

func (m Metric) IsValid() bool {
	switch m {
	case MetricCount, MetricPriceMin, MetricPriceMax, MetricBrandCount, MetricCategoryCount:
		return true
	default:
		return false
	}
}

// VariableFilter scopes a variable to a slice of the catalog using slugs —
// the stable, human-meaningful identifiers content authors pick from an
// editor UI, never raw database IDs. GroupSlug expands to every category
// under that group (e.g. every installation-type category under
// "may-lanh"); CategorySlug narrows to one. BrandSlug further narrows
// either to one brand. At least one of GroupSlug/CategorySlug is required.
type VariableFilter struct {
	GroupSlug    string
	CategorySlug string
	BrandSlug    string
}

// VariableRequest is one {id, metric, filter} entry in a batch resolve
// call. ID is caller-assigned (the Tiptap node's own key) so the response
// map can be matched back to the node that asked for it.
type VariableRequest struct {
	ID     string
	Metric Metric
	Filter VariableFilter
}
