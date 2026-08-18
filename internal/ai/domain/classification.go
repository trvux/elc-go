package domain

// ClassificationResult is the guardrail classifier's verdict on one
// customer message, run before the (more expensive) main chat completion —
// see internal/ai/application's classify_message.go.
type ClassificationResult struct {
	OnTopic bool
	Reason  string
}
