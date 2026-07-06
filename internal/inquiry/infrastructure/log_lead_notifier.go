package infrastructure

import (
	"context"

	"go.uber.org/zap"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

// LogLeadNotifier is the fallback LeadNotifier used until Zalo OA credentials
// are configured (see ZALO_OA_APP_ID/ZALO_OA_APP_SECRET in cmd/server/main.go).
// It logs the lead instead of pushing a notification, so lead capture works
// end-to-end from day one — staff see new leads in /admin/inquiries even
// before Zalo is wired up. Mirrors auth.LogEmailSender's role exactly.
type LogLeadNotifier struct {
	log *zap.Logger
}

var _ domain.LeadNotifier = (*LogLeadNotifier)(nil)

func NewLogLeadNotifier(log *zap.Logger) *LogLeadNotifier {
	return &LogLeadNotifier{log: log}
}

func (n *LogLeadNotifier) NotifyNewLead(ctx context.Context, inquiry *domain.Inquiry) error {
	n.log.Info("new lead received (Zalo OA not configured — logging instead of notifying)",
		zap.String("id", inquiry.ID()),
		zap.String("name", inquiry.Name()),
		zap.String("phone", inquiry.Phone()),
	)
	return nil
}
