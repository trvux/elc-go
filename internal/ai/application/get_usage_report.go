package application

import (
	"context"
	"time"

	"github.com/trvux/elc-go/internal/ai/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

func GetUsageReport(ctx context.Context, repo domain.ConversationRepository, from, to time.Time, groupBy domain.UsageGroupBy) ([]domain.UsageReportRow, error) {
	if !groupBy.IsValid() {
		return nil, apperr.NewValidationError("validation failed", map[string][]string{"groupBy": {`must be "day", "provider", or "model"`}})
	}
	return repo.GetUsageReport(ctx, from, to, groupBy)
}
