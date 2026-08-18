package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/trvux/elc-go/internal/ai/domain"
)

// ToolExecutor runs one tool call and returns its result (typically JSON) to
// feed back to the model as a RoleTool message.
type ToolExecutor func(ctx context.Context, argumentsJSON string) (string, error)

// Tool pairs a domain.ToolDefinition (what the model sees) with the
// executor that runs it (what actually happens when the model calls it).
type Tool struct {
	Definition domain.ToolDefinition
	Execute    ToolExecutor
}

// maxToolRounds bounds the tool-call loop within one model attempt — a
// model that keeps requesting tools must not turn one customer message into
// unbounded API spend or an effectively-hung request.
const maxToolRounds = 3

// SystemPrompt is the assistant's persona and ground rules, prepended to
// every conversation.
const SystemPrompt = `Bạn là trợ lý tư vấn bán hàng của ELC (cửa hàng điện máy). ` +
	`Trả lời ngắn gọn, thân thiện, bằng tiếng Việt. ` +
	`Luôn dùng công cụ search_products để tra cứu trước khi nhắc tên hoặc giá một sản phẩm cụ thể — ` +
	`không tự bịa tên sản phẩm, giá, hay tình trạng còn hàng. ` +
	`Nếu không tìm thấy sản phẩm phù hợp, nói thật là chưa có, đừng đoán bừa. ` +
	`Chỉ tư vấn về sản phẩm/dịch vụ của ELC, không nhắc tên hay so sánh với cửa hàng điện máy khác.`

// ChatOutcome is what SendChatMessageStream settles on: the assistant's
// final reply, which ModelConfig actually answered (after any fallback),
// and the token usage that answer billed. The caller (presentation) uses
// Model to compute domain.Pricing.Cost and persist provider_id/model_id,
// and ProductsShown to persist products_shown for the ads/analytics goal.
type ChatOutcome struct {
	Message       domain.Message
	Model         domain.ModelConfig
	Usage         domain.TokenUsage
	ProductsShown []string
}

// SendChatMessageStream tries chatModels in order (fallback_priority
// ascending — see ResolveActiveModels), running the tool-call loop against
// the first one that doesn't fail with a retryable error (429/timeout/5xx —
// domain.RetryableError), forwarding content deltas to emit as they arrive.
//
// Once any delta has been forwarded for a given model attempt, that
// attempt's errors are no longer treated as retryable regardless of what
// the transport itself reports — a caller that's already streamed partial
// text to its own client cannot cleanly switch to a different model's
// answer (see the Phase 2 RFC). A non-retryable error aborts the whole
// fallback loop immediately, same as it would fail identically against
// every other model — but the returned *ChatOutcome is still non-nil
// whenever some text was already streamed to the client before the error,
// carrying that partial reply in Message.Content so the caller (presentation)
// can persist it (marked incomplete) instead of silently losing it. A
// retryable failure (nothing streamed yet) always returns a nil outcome —
// there is nothing partial to keep in that case.
func SendChatMessageStream(ctx context.Context, clientFactory domain.LLMClientFactory, chatModels []domain.ModelConfig, tools []Tool, history []domain.Message, emit func(delta string)) (*ChatOutcome, error) {
	if len(chatModels) == 0 {
		return nil, fmt.Errorf("ai: no active chat model configured")
	}

	var lastErr error
	for _, cfg := range chatModels {
		outcome, err := streamWithModel(ctx, clientFactory(cfg), cfg, tools, history, emit)
		if err == nil {
			return outcome, nil
		}
		lastErr = err
		if !isRetryable(err) {
			return outcome, err
		}
	}
	return nil, fmt.Errorf("ai: every configured chat model failed, last error: %w", lastErr)
}

// streamWithModel returns a non-nil *ChatOutcome even on error whenever
// content was already streamed for this attempt (see streamOnce) — the
// error itself always still propagates, callers must check both.
func streamWithModel(ctx context.Context, client domain.LLMClient, cfg domain.ModelConfig, tools []Tool, history []domain.Message, emit func(string)) (*ChatOutcome, error) {
	toolDefs, executors := splitTools(tools)
	messages := append([]domain.Message{{Role: domain.RoleSystem, Content: SystemPrompt}}, history...)

	var usage domain.TokenUsage
	var productsShown []string
	emittedAny := false

	for round := 0; round < maxToolRounds; round++ {
		content, final, err := streamOnce(ctx, client, messages, toolDefs, emit, &emittedAny)
		if err != nil {
			return partialOutcome(content, cfg, usage, productsShown), err
		}
		usage = addUsage(usage, final.Usage)

		if len(final.ToolCalls) == 0 {
			return &ChatOutcome{Message: domain.Message{Role: domain.RoleAssistant, Content: content}, Model: cfg, Usage: usage, ProductsShown: productsShown}, nil
		}

		toolResults := runTools(ctx, executors, final.ToolCalls)
		productsShown = append(productsShown, extractProductSlugs(final.ToolCalls, toolResults)...)
		messages = append(messages, domain.Message{Role: domain.RoleAssistant, Content: content, ToolCalls: final.ToolCalls})
		messages = append(messages, toolResults...)
	}

	// Round budget exceeded — one forced final call with tools disabled,
	// still streamed so whatever text it produces still reaches the client
	// in real time.
	content, final, err := streamOnce(ctx, client, messages, nil, emit, &emittedAny)
	if err != nil {
		return partialOutcome(content, cfg, usage, productsShown), err
	}
	usage = addUsage(usage, final.Usage)
	return &ChatOutcome{Message: domain.Message{Role: domain.RoleAssistant, Content: content}, Model: cfg, Usage: usage, ProductsShown: productsShown}, nil
}

// partialOutcome wraps whatever content a failed round managed to stream
// before erroring — nil (not an empty-content outcome) when there's
// nothing to keep, so callers can tell "partial reply to persist" apart
// from "nothing was ever shown" with a single nil check.
func partialOutcome(content string, cfg domain.ModelConfig, usage domain.TokenUsage, productsShown []string) *ChatOutcome {
	if content == "" {
		return nil
	}
	return &ChatOutcome{Message: domain.Message{Role: domain.RoleAssistant, Content: content}, Model: cfg, Usage: usage, ProductsShown: productsShown}
}

// streamOnce runs a single ChatStream round, forwarding content deltas to
// emit and returning the accumulated content plus the round's final event.
// The accumulated content is returned even when it ends in an error — a
// round that streamed several sentences and then failed still has that
// text to hand back, see partialOutcome. *emittedAny tracks whether any
// delta has been forwarded across the whole model attempt (not just this
// round) — once true, any error from here on is stripped of its
// RetryableError wrapper via stripRetryable, so SendChatMessageStream's
// fallback loop won't try another model after the client has already seen
// partial text.
func streamOnce(ctx context.Context, client domain.LLMClient, messages []domain.Message, toolDefs []domain.ToolDefinition, emit func(string), emittedAny *bool) (string, domain.StreamEvent, error) {
	events, err := client.ChatStream(ctx, messages, toolDefs)
	if err != nil {
		// ChatStream itself failed before any events — nothing was ever
		// streamed for this call, so there is no content to return.
		if *emittedAny {
			return "", domain.StreamEvent{}, stripRetryable(err)
		}
		return "", domain.StreamEvent{}, err
	}

	var content strings.Builder
	var final domain.StreamEvent
	for ev := range events {
		if ev.ContentDelta != "" {
			content.WriteString(ev.ContentDelta)
			emit(ev.ContentDelta)
			*emittedAny = true
		}
		if ev.Done {
			final = ev
		}
	}

	if final.Err != nil {
		if *emittedAny {
			return content.String(), domain.StreamEvent{}, stripRetryable(final.Err)
		}
		return content.String(), domain.StreamEvent{}, final.Err
	}
	return content.String(), final, nil
}

func stripRetryable(err error) error {
	var retryable *domain.RetryableError
	if errors.As(err, &retryable) {
		return retryable.Err
	}
	return err
}

func addUsage(a, b domain.TokenUsage) domain.TokenUsage {
	return domain.TokenUsage{
		InputTokens:    a.InputTokens + b.InputTokens,
		OutputTokens:   a.OutputTokens + b.OutputTokens,
		CacheHitTokens: a.CacheHitTokens + b.CacheHitTokens,
	}
}

// isRetryable reports whether err (or something it wraps) is a
// domain.RetryableError — the shared fallback-vs-abort test used by
// SendChatMessageStream and ClassifyMessage.
func isRetryable(err error) bool {
	var retryable *domain.RetryableError
	return errors.As(err, &retryable)
}

func splitTools(tools []Tool) ([]domain.ToolDefinition, map[string]ToolExecutor) {
	defs := make([]domain.ToolDefinition, len(tools))
	executors := make(map[string]ToolExecutor, len(tools))
	for i, t := range tools {
		defs[i] = t.Definition
		executors[t.Definition.Name] = t.Execute
	}
	return defs, executors
}

func runTools(ctx context.Context, executors map[string]ToolExecutor, calls []domain.ToolCall) []domain.Message {
	results := make([]domain.Message, len(calls))
	for i, tc := range calls {
		exec, ok := executors[tc.Name]
		var content string
		switch {
		case !ok:
			content = fmt.Sprintf(`{"error":"unknown tool %q"}`, tc.Name)
		default:
			out, err := exec(ctx, tc.Arguments)
			if err != nil {
				content = fmt.Sprintf(`{"error":%q}`, err.Error())
			} else {
				content = out
			}
		}
		results[i] = domain.Message{Role: domain.RoleTool, Content: content, ToolCallID: tc.ID}
	}
	return results
}

// extractProductSlugs pulls product slugs out of search_products tool
// results, purely to populate products_shown for the ads/analytics goal
// (see domain.ConversationMessage's doc comment) — a small, deliberate
// coupling to that one tool's output shape rather than a generic
// "any tool can report analytics" mechanism nothing else needs yet.
func extractProductSlugs(calls []domain.ToolCall, results []domain.Message) []string {
	var slugs []string
	for i, tc := range calls {
		if tc.Name != "search_products" || i >= len(results) {
			continue
		}
		var items []struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal([]byte(results[i].Content), &items); err != nil {
			continue
		}
		for _, it := range items {
			if it.Slug != "" {
				slugs = append(slugs, it.Slug)
			}
		}
	}
	return slugs
}
