package httpserver

import (
	"net/http"
	"strconv"

	"github.com/trvux/elc-go/internal/platform/apperr"
)

// ParsePagination reads and validates the "limit"/"offset" query params
// shared by every list endpoint's filter. Omitting either param leaves it at
// 0 — several list handlers intentionally treat Limit<=0 as "no limit" (a
// dropdown/select that wants the full set, not a paginated page), so this
// only rejects a param that IS present but invalid (non-numeric, or a
// negative/zero value), rather than requiring both params on every call.
func ParsePagination(r *http.Request) (limit, offset int, err error) {
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit <= 0 {
			return 0, 0, apperr.NewValidationError("validation failed", map[string][]string{
				"limit": {"limit must be a positive integer"},
			})
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return 0, 0, apperr.NewValidationError("validation failed", map[string][]string{
				"offset": {"offset must be a non-negative integer"},
			})
		}
	}
	return limit, offset, nil
}
