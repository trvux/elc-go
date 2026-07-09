package application

import (
	"context"

	"github.com/trvux/elc-go/internal/product/domain"
)

// GetAdjacentProducts is a thin passthrough — see
// internal/product/infrastructure/postgres_repository.go's GetAdjacent for
// where the actual "same-category siblings first, fall back to the full
// published catalog when fewer than 2 published siblings exist" decision
// lives.
//
// Design note (documented per the approved plan): the old TS
// getAdjacentProducts.ts called getProducts(...) up to twice — once scoped
// to the category, then again unscoped as a fallback — entirely in the
// application layer, each call a full repository round trip. Pushing both
// queries into GetAdjacent (single method, same repository call from here)
// turns that into one SQL round trip instead of two, which matters more here
// than it did for service's getAdjacentServices (same pattern, not yet
// pushed down) because this list can be much larger (196+ products vs. a
// handful of services). The trade-off: the "< 2 siblings -> broaden scope"
// branch itself is only exercised by the infrastructure integration test
// (real category data), not a fake-repository unit test here — a fake
// repository's GetAdjacent is just a stub returning whatever the test wires
// up, so there is no separate branching logic at this layer worth unit
// testing in isolation.
func GetAdjacentProducts(ctx context.Context, repo domain.ProductRepository, categoryID, currentID string) (prev, next *domain.AdjacentProduct, err error) {
	return repo.GetAdjacent(ctx, categoryID, currentID)
}
