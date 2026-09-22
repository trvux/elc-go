package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*MeasurementProtocolClient, func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	client := &MeasurementProtocolClient{
		measurementID: "G-TEST123",
		apiSecret:     "test-secret",
		baseURL:       server.URL,
		httpClient:    &http.Client{Timeout: 2 * time.Second},
	}
	return client, server.Close
}

func TestMeasurementProtocolClient_IsConfigured(t *testing.T) {
	cases := []struct {
		name          string
		measurementID string
		apiSecret     string
		want          bool
	}{
		{"both set", "G-TEST123", "secret", true},
		{"missing measurement id", "", "secret", false},
		{"missing api secret", "G-TEST123", "", false},
		{"both missing", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewMeasurementProtocolClient(tc.measurementID, tc.apiSecret)
			if got := client.IsConfigured(); got != tc.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMeasurementProtocolClient_SendCloseConvertLead_SendsCorrectPayload(t *testing.T) {
	var capturedQuery string
	var capturedBody mpPayload

	client, closeServer := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeServer()

	value := 15_500_000.0
	err := client.SendCloseConvertLead(context.Background(), "1234567890.9876543210", &value)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if capturedQuery != "measurement_id=G-TEST123&api_secret=test-secret" {
		t.Errorf("unexpected query string: %q", capturedQuery)
	}
	if capturedBody.ClientID != "1234567890.9876543210" {
		t.Errorf("expected client_id to be the ga_client_id passed in, got %q", capturedBody.ClientID)
	}
	if len(capturedBody.Events) != 1 || capturedBody.Events[0].Name != "close_convert_lead" {
		t.Fatalf("expected exactly 1 close_convert_lead event, got %+v", capturedBody.Events)
	}
	gotValue, ok := capturedBody.Events[0].Params["value"].(float64)
	if !ok || gotValue != value {
		t.Errorf("expected params.value = %v, got %v", value, capturedBody.Events[0].Params["value"])
	}
	if capturedBody.Events[0].Params["currency"] != "VND" {
		t.Errorf("expected params.currency = VND, got %v", capturedBody.Events[0].Params["currency"])
	}
}

func TestMeasurementProtocolClient_SendCloseConvertLead_OmitsValueWhenNil(t *testing.T) {
	var capturedBody mpPayload

	client, closeServer := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeServer()

	if err := client.SendCloseConvertLead(context.Background(), "1234567890.9876543210", nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, hasValue := capturedBody.Events[0].Params["value"]; hasValue {
		t.Error("expected no value param when staff didn't record an order value")
	}
}

func TestMeasurementProtocolClient_SendCloseConvertLead_RejectsBlankGAClientID(t *testing.T) {
	client, closeServer := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("expected no HTTP request to be made without a ga_client_id")
	})
	defer closeServer()

	if err := client.SendCloseConvertLead(context.Background(), "", nil); err == nil {
		t.Fatal("expected an error for a blank ga_client_id")
	}
}

func TestMeasurementProtocolClient_SendCloseConvertLead_ErrorsOnNon2xx(t *testing.T) {
	client, closeServer := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	defer closeServer()

	if err := client.SendCloseConvertLead(context.Background(), "1234567890.9876543210", nil); err == nil {
		t.Fatal("expected an error when Measurement Protocol rejects the request")
	}
}

func TestMeasurementProtocolClient_SendCloseConvertLead_RejectsWhenNotConfigured(t *testing.T) {
	client := NewMeasurementProtocolClient("", "")
	if err := client.SendCloseConvertLead(context.Background(), "1234567890.9876543210", nil); err == nil {
		t.Fatal("expected an error when the client has no measurement id/api secret")
	}
}

func TestNoopNotifier(t *testing.T) {
	notifier := NewNoopNotifier()
	if notifier.IsConfigured() {
		t.Error("expected NoopNotifier.IsConfigured() to always be false")
	}
	if err := notifier.SendCloseConvertLead(context.Background(), "1234567890.9876543210", nil); err == nil {
		t.Error("expected NoopNotifier.SendCloseConvertLead to always error")
	}
}
