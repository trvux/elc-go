package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/trvux/elc-go/internal/adsconversion/domain"
)

const measurementProtocolURL = "https://www.google-analytics.com/mp/collect"

// MeasurementProtocolClient pushes GA4 key events server-side, via
// Google's Measurement Protocol — chosen over a direct Google Ads offline
// conversion upload (see docs/rfc for the measurement-loop plan) because
// close_convert_lead/qualify_lead are already configured as GA4-imported
// primary conversion actions in Ads: this reuses that existing link
// instead of standing up a separate Ads-side conversion action, and needs
// only a static API secret (no OAuth2), which is far simpler to run from a
// Go backend than the full Google Ads API client.
type MeasurementProtocolClient struct {
	measurementID string
	apiSecret     string
	// baseURL defaults to measurementProtocolURL — overridable only by
	// tests in this package (an httptest.Server), never by callers outside
	// it: there's exactly one real Measurement Protocol endpoint.
	baseURL    string
	httpClient *http.Client
}

var _ domain.Notifier = (*MeasurementProtocolClient)(nil)

func NewMeasurementProtocolClient(measurementID, apiSecret string) *MeasurementProtocolClient {
	return &MeasurementProtocolClient{
		measurementID: measurementID,
		apiSecret:     apiSecret,
		baseURL:       measurementProtocolURL,
		// Short — this blocks the admin's "Lưu thay đổi" click (see
		// application.UpdateInquiryStatus), so a hung request to Google
		// must not stall the request for long.
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *MeasurementProtocolClient) IsConfigured() bool {
	return c != nil && c.measurementID != "" && c.apiSecret != ""
}

type mpEvent struct {
	Name   string         `json:"name"`
	Params map[string]any `json:"params"`
}

type mpPayload struct {
	ClientID string    `json:"client_id"`
	Events   []mpEvent `json:"events"`
}

func (c *MeasurementProtocolClient) SendCloseConvertLead(ctx context.Context, gaClientID string, value *float64) error {
	if !c.IsConfigured() {
		return fmt.Errorf("measurement protocol client: not configured (GA4_MEASUREMENT_ID/GA4_API_SECRET unset)")
	}
	if gaClientID == "" {
		return fmt.Errorf("measurement protocol client: no ga_client_id to attribute this event to")
	}

	params := map[string]any{}
	if value != nil {
		params["value"] = *value
		params["currency"] = "VND"
	}

	body, err := json.Marshal(mpPayload{
		ClientID: gaClientID,
		Events:   []mpEvent{{Name: "close_convert_lead", Params: params}},
	})
	if err != nil {
		return fmt.Errorf("measurement protocol client: marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s?measurement_id=%s&api_secret=%s", c.baseURL, c.measurementID, c.apiSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("measurement protocol client: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("measurement protocol client: send request: %w", err)
	}
	defer resp.Body.Close()

	// Measurement Protocol returns 204 on success — but ALSO on a
	// malformed/silently-dropped event; there is no server-side validation
	// response, Google only surfaces shape mistakes via GA4 DebugView. A
	// non-2xx here means the request itself was rejected outright (bad
	// measurement_id/api_secret, network-level issue) — still treated as
	// an error since the caller uses it to decide whether to record
	// ads_conversion_synced_at, but a 2xx is NOT a guarantee GA4 actually
	// recorded the event, only that Google accepted the HTTP request.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("measurement protocol client: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// NoopNotifier lets the server boot without GA4 credentials configured
// (e.g. local dev) — IsConfigured() false means callers skip it entirely,
// so SendCloseConvertLead here only exists to satisfy the interface and is
// never actually reached in practice. Same posture as internal/upload's
// NoopUploader.
type NoopNotifier struct{}

var _ domain.Notifier = (*NoopNotifier)(nil)

func NewNoopNotifier() *NoopNotifier { return &NoopNotifier{} }

func (n *NoopNotifier) IsConfigured() bool { return false }

func (n *NoopNotifier) SendCloseConvertLead(ctx context.Context, gaClientID string, value *float64) error {
	return fmt.Errorf("ads conversion sync is not configured: set GA4_MEASUREMENT_ID/GA4_API_SECRET")
}
