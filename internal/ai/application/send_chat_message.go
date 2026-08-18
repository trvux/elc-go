package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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

// ChatOutcome is what SendChatMessage settles on: the assistant's final
// reply, which ModelConfig actually answered (after any fallback), and the
// token usage that answer billed. The caller (presentation) uses Model to
// compute domain.Pricing.Cost and persist provider_id/model_id, and
// ProductsShown to persist products_shown for the ads/analytics goal.
type ChatOutcome struct {
	Message       domain.Message
	Model         domain.ModelConfig
	Usage         domain.TokenUsage
	ProductsShown []string
}

// SendChatMessage tries chatModels in order (fallback_priority ascending —
// see ResolveActiveModels), running the tool-call loop against the first
// one that doesn't fail with a retryable error (429/timeout/5xx — see
// domain.RetryableError's doc comment). A non-retryable error aborts
// immediately instead of working through the rest of the chain: it would
// fail identically against every other model too.
func SendChatMessage(ctx context.Context, clientFactory domain.LLMClientFactory, chatModels []domain.ModelConfig, tools []Tool, history []domain.Message) (*ChatOutcome, error) {
	if len(chatModels) == 0 {
		return nil, fmt.Errorf("ai: no active chat model configured")
	}

	var lastErr error
	for _, cfg := range chatModels {
		outcome, err := sendWithModel(ctx, clientFactory(cfg), cfg, tools, history)
		if err == nil {
			return outcome, nil
		}
		lastErr = err
		if !isRetryable(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("ai: every configured chat model failed, last error: %w", lastErr)
}

func sendWithModel(ctx context.Context, client domain.LLMClient, cfg domain.ModelConfig, tools []Tool, history []domain.Message) (*ChatOutcome, error) {
	toolDefs, executors := splitTools(tools)
	messages := append([]domain.Message{{Role: domain.RoleSystem, Content: SystemPrompt}}, history...)

	var usage domain.TokenUsage
	var productsShown []string

	for round := 0; round < maxToolRounds; round++ {
		result, err := client.Chat(ctx, messages, toolDefs)
		if err != nil {
			return nil, err
		}
		usage = addUsage(usage, result.Usage)
		if len(result.Message.ToolCalls) == 0 {
			return &ChatOutcome{Message: result.Message, Model: cfg, Usage: usage, ProductsShown: productsShown}, nil
		}

		toolResults := runTools(ctx, executors, result.Message.ToolCalls)
		productsShown = append(productsShown, extractProductSlugs(result.Message.ToolCalls, toolResults)...)

		messages = append(messages, result.Message)
		messages = append(messages, toolResults...)
	}

	// Model kept requesting tools past the round budget — ask once more
	// with tools disabled so it must answer from what it already has.
	final, err := client.Chat(ctx, messages, nil)
	if err != nil {
		return nil, err
	}
	usage = addUsage(usage, final.Usage)
	return &ChatOutcome{Message: final.Message, Model: cfg, Usage: usage, ProductsShown: productsShown}, nil
}

func addUsage(a, b domain.TokenUsage) domain.TokenUsage {
	return domain.TokenUsage{
		InputTokens:    a.InputTokens + b.InputTokens,
		OutputTokens:   a.OutputTokens + b.OutputTokens,
		CacheHitTokens: a.CacheHitTokens + b.CacheHitTokens,
	}
}

// isRetryable reports whether err (or something it wraps) is a
// domain.RetryableError — the shared fallback-vs-abort test used by both
// SendChatMessage and ClassifyMessage.
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
