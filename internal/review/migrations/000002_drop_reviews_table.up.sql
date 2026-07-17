-- The customer-review feature was removed entirely (frontend and Go module
-- both deleted — see internal/review's now-empty domain/application/
-- infrastructure/presentation). reviews is a fully dedicated table (no other
-- module's schema references it), so dropping it is a clean, isolated
-- operation — no ALTER on products/services needed.
DROP TABLE IF EXISTS reviews;
