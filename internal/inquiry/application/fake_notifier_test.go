package application

import "context"

// fakeNotifier is an in-memory stand-in for adsconversion infrastructure's
// MeasurementProtocolClient, used only in tests.
type fakeNotifier struct {
	configured bool
	failWith   error
	calls      []fakeNotifierCall
}

type fakeNotifierCall struct {
	gaClientID string
	value      *float64
}

func (n *fakeNotifier) IsConfigured() bool { return n.configured }

func (n *fakeNotifier) SendCloseConvertLead(ctx context.Context, gaClientID string, value *float64) error {
	n.calls = append(n.calls, fakeNotifierCall{gaClientID: gaClientID, value: value})
	return n.failWith
}
