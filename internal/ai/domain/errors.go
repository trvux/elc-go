package domain

// RetryableError wraps an LLMClient error that should trigger fallback to
// the next model in the chain — a 429 (DeepSeek's docs: rate-limited by
// account-wide concurrency, not requests/minute), a timeout, or a 5xx. A
// plain (non-retryable) error — e.g. a 400 from a malformed payload — would
// fail identically against every provider, so SendChatMessage's fallback
// loop aborts immediately on those instead of wasting the remaining
// attempts.
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }
