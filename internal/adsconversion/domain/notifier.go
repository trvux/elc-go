package domain

import "context"

// Notifier pushes a "real sale closed" signal to GA4/Google Ads once staff
// mark a lead converted — implemented by infrastructure (a GA4 Measurement
// Protocol client). Defined here, not in internal/inquiry, so that module
// depends on this interface rather than a concrete HTTP client — same
// reasoning as internal/upload/domain's Uploader interface.
type Notifier interface {
	// IsConfigured reports whether the credentials needed to actually send
	// anything are present — callers should skip the push (not error) when
	// this is false, same posture as internal/upload's NoopUploader for
	// missing R2 credentials, except this one fails silently (logged, not
	// surfaced) since a missing GA4 secret must never block staff from
	// marking a lead converted.
	IsConfigured() bool
	// SendCloseConvertLead pushes the close_convert_lead GA4 key event,
	// attributed to gaClientID (the visitor's own GA4 client id, captured
	// at lead-creation time — see internal/inquiry's ga_client_id column).
	// value is the order amount in VND, nil if staff didn't record one.
	SendCloseConvertLead(ctx context.Context, gaClientID string, value *float64) error
}
