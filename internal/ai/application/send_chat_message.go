package application

import (
	"context"
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

// maxToolRounds bounds the tool-call loop in SendChatMessage — a model that
// keeps requesting tools must not turn one customer message into unbounded
// API spend or an effectively-hung request.
const maxToolRounds = 3

// SystemPrompt is the assistant's persona and ground rules, prepended to
// every conversation.
const SystemPrompt = `Bạn là trợ lý tư vấn bán hàng của ELC (cửa hàng điện máy). ` +
	`Trả lời ngắn gọn, thân thiện, bằng tiếng Việt. ` +
	`Luôn dùng công cụ search_products để tra cứu trước khi nhắc tên hoặc giá một sản phẩm cụ thể — ` +
	`không tự bịa tên sản phẩm, giá, hay tình trạng còn hàng. ` +
	`Nếu không tìm thấy sản phẩm phù hợp, nói thật là chưa có, đừng đoán bừa.`

// SendChatMessage sends the conversation (history plus the customer's
// latest message) to client, running any tool calls the model requests
// (up to maxToolRounds rounds) before returning its final reply.
func SendChatMessage(ctx context.Context, client domain.LLMClient, tools []Tool, history []domain.Message) (*domain.Message, error) {
	toolDefs, executors := splitTools(tools)
	messages := append([]domain.Message{{Role: domain.RoleSystem, Content: SystemPrompt}}, history...)

	for round := 0; round < maxToolRounds; round++ {
		result, err := client.Chat(ctx, messages, toolDefs)
		if err != nil {
			return nil, fmt.Errorf("ai: chat: %w", err)
		}
		if len(result.Message.ToolCalls) == 0 {
			return &result.Message, nil
		}

		messages = append(messages, result.Message)
		messages = append(messages, runTools(ctx, executors, result.Message.ToolCalls)...)
	}

	// Model kept requesting tools past the round budget — ask once more
	// with tools disabled so it must answer from what it already has.
	final, err := client.Chat(ctx, messages, nil)
	if err != nil {
		return nil, fmt.Errorf("ai: chat (final): %w", err)
	}
	return &final.Message, nil
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
