// Package infrastructure implements internal/ai/domain's ports: the
// LLMClient against an OpenAI-compatible chat-completions API, and the
// product-search tool the model can call.
package infrastructure

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/ai/domain"
)

// OpenAICompatibleClient is a domain.LLMClient for any provider that speaks
// the OpenAI chat-completions wire format — DeepSeek today, and (by pointing
// BaseURL/Model/APIKey at a different provider, no code change) GLM,
// Moonshot, Qwen, Groq, etc. later.
type OpenAICompatibleClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenAICompatibleClient builds a client against baseURL (e.g.
// "https://api.deepseek.com/v1") using model (e.g. "deepseek-chat").
// httpTimeout backstops a caller that forgets to put a deadline on ctx —
// every real call should still pass a context.WithTimeout of its own (see
// golang-design-patterns: every external call needs a timeout).
func NewOpenAICompatibleClient(baseURL, apiKey, model string, httpTimeout time.Duration) *OpenAICompatibleClient {
	return &OpenAICompatibleClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

var _ domain.LLMClient = (*OpenAICompatibleClient)(nil)

// llmClientHTTPTimeout backstops every OpenAICompatibleClient built by
// NewLLMClient — see NewOpenAICompatibleClient's doc comment on why this
// isn't the caller's only timeout.
const llmClientHTTPTimeout = 30 * time.Second

// NewLLMClient adapts NewOpenAICompatibleClient to domain.LLMClientFactory
// — the shape SendChatMessage/ClassifyMessage's fallback loops use to build
// a fresh client per resolved ModelConfig. Composition root
// (cmd/server/main.go) wires this in directly: `aiInfra.NewLLMClient`.
func NewLLMClient(cfg domain.ModelConfig) domain.LLMClient {
	return NewOpenAICompatibleClient(cfg.BaseURL, cfg.APIKey, cfg.ModelName, llmClientHTTPTimeout)
}

// --- wire shapes (OpenAI chat-completions request/response) ---

type wireMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type wireToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type wireTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type wireStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type wireRequest struct {
	Model         string             `json:"model"`
	Messages      []wireMessage      `json:"messages"`
	Tools         []wireTool         `json:"tools,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	StreamOptions *wireStreamOptions `json:"stream_options,omitempty"`
}

// wireStreamChunk is one `data: {...}` line of an SSE chat-completions
// stream. ToolCalls arrive as index-keyed fragments — id and the function
// name typically land whole on the first fragment for a given index,
// arguments arrive character-by-character across many fragments — see
// readStream's accumulation.
type wireStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens         int `json:"prompt_tokens"`
		CompletionTokens     int `json:"completion_tokens"`
		PromptCacheHitTokens int `json:"prompt_cache_hit_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type wireResponse struct {
	Choices []struct {
		Message wireMessage `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens         int `json:"prompt_tokens"`
		CompletionTokens     int `json:"completion_tokens"`
		PromptCacheHitTokens int `json:"prompt_cache_hit_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Chat implements domain.LLMClient.
func (c *OpenAICompatibleClient) Chat(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition) (*domain.ChatResult, error) {
	reqBody := wireRequest{
		Model:    c.model,
		Messages: toWireMessages(messages),
		Tools:    toWireTools(tools),
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ai: encode chat request: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ai: build chat request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Network failure or timeout — the caller's fallback loop should
		// try the next model, same as a 429/5xx below.
		return nil, &domain.RetryableError{Err: fmt.Errorf("ai: call chat completions: %w", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ai: read chat response: %w", err)
	}

	var wireResp wireResponse
	if err := json.Unmarshal(body, &wireResp); err != nil {
		return nil, fmt.Errorf("ai: decode chat response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		providerErr := fmt.Errorf("ai: provider returned %d", resp.StatusCode)
		if wireResp.Error != nil {
			providerErr = fmt.Errorf("ai: provider returned %d: %s", resp.StatusCode, wireResp.Error.Message)
		}
		// 429 (DeepSeek's docs: rate-limited by account-wide concurrency,
		// not requests/minute) and 5xx are retryable against the next
		// model in the fallback chain — any other 4xx (e.g. a malformed
		// payload) would fail identically everywhere, so it's returned
		// plain and SendChatMessage aborts on it instead of wasting the
		// rest of the chain.
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			return nil, &domain.RetryableError{Err: providerErr}
		}
		return nil, providerErr
	}
	if len(wireResp.Choices) == 0 {
		return nil, &domain.RetryableError{Err: fmt.Errorf("ai: provider returned no choices")}
	}

	result := &domain.ChatResult{Message: fromWireMessage(wireResp.Choices[0].Message)}
	if wireResp.Usage != nil {
		result.Usage = domain.TokenUsage{
			InputTokens:    wireResp.Usage.PromptTokens,
			OutputTokens:   wireResp.Usage.CompletionTokens,
			CacheHitTokens: wireResp.Usage.PromptCacheHitTokens,
		}
	}
	return result, nil
}

// ChatStream implements domain.LLMClient. It makes the HTTP request and
// waits for headers/status synchronously — same retryable-vs-plain error
// classification as Chat — so a caller's fallback loop can still switch to
// the next model on a pre-stream failure. Only once the response is
// confirmed 200 OK does it hand off to a goroutine reading the SSE body;
// errors past that point arrive as the final StreamEvent (Err set) instead,
// since nothing about a already-flushed-to-the-caller stream can be retried
// (see the Phase 2 RFC's fallback-only-before-first-byte rule).
func (c *OpenAICompatibleClient) ChatStream(ctx context.Context, messages []domain.Message, tools []domain.ToolDefinition) (<-chan domain.StreamEvent, error) {
	reqBody := wireRequest{
		Model:         c.model,
		Messages:      toWireMessages(messages),
		Tools:         toWireTools(tools),
		Stream:        true,
		StreamOptions: &wireStreamOptions{IncludeUsage: true},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ai: encode chat stream request: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ai: build chat stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, &domain.RetryableError{Err: fmt.Errorf("ai: call chat completions (stream): %w", err)}
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var wireResp wireResponse
		_ = json.Unmarshal(body, &wireResp)

		providerErr := fmt.Errorf("ai: provider returned %d", resp.StatusCode)
		if wireResp.Error != nil {
			providerErr = fmt.Errorf("ai: provider returned %d: %s", resp.StatusCode, wireResp.Error.Message)
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			return nil, &domain.RetryableError{Err: providerErr}
		}
		return nil, providerErr
	}

	events := make(chan domain.StreamEvent)
	go readStream(resp.Body, events)
	return events, nil
}

// readStream parses an SSE chat-completions body, forwarding each content
// delta immediately and accumulating tool-call fragments (id/name land
// whole, arguments accumulate character-by-character — index-keyed, same
// as the OpenAI streaming tool-call convention) until the final `data:
// [DONE]` line, at which point it emits one Done event carrying the
// accumulated tool calls and usage. Always closes events before returning.
func readStream(body io.ReadCloser, events chan<- domain.StreamEvent) {
	defer close(events)
	defer body.Close()

	type toolCallBuilder struct {
		id, name, arguments string
	}
	builders := make(map[int]*toolCallBuilder)
	maxIndex := -1
	var usage domain.TokenUsage

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		data, ok := strings.CutPrefix(scanner.Text(), "data: ")
		if !ok || data == "" {
			continue
		}
		if data == "[DONE]" {
			break
		}

		var chunk wireStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			events <- domain.StreamEvent{Done: true, Err: fmt.Errorf("ai: decode stream chunk: %w", err)}
			return
		}
		if chunk.Error != nil {
			events <- domain.StreamEvent{Done: true, Err: fmt.Errorf("ai: provider stream error: %s", chunk.Error.Message)}
			return
		}
		if chunk.Usage != nil {
			usage = domain.TokenUsage{
				InputTokens:    chunk.Usage.PromptTokens,
				OutputTokens:   chunk.Usage.CompletionTokens,
				CacheHitTokens: chunk.Usage.PromptCacheHitTokens,
			}
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		if delta.Content != "" {
			events <- domain.StreamEvent{ContentDelta: delta.Content}
		}
		for _, tc := range delta.ToolCalls {
			b, ok := builders[tc.Index]
			if !ok {
				b = &toolCallBuilder{}
				builders[tc.Index] = b
				if tc.Index > maxIndex {
					maxIndex = tc.Index
				}
			}
			if tc.ID != "" {
				b.id = tc.ID
			}
			if tc.Function.Name != "" {
				b.name = tc.Function.Name
			}
			b.arguments += tc.Function.Arguments
		}
	}

	if err := scanner.Err(); err != nil {
		events <- domain.StreamEvent{Done: true, Err: fmt.Errorf("ai: read stream: %w", err)}
		return
	}

	var toolCalls []domain.ToolCall
	for i := 0; i <= maxIndex; i++ {
		if b, ok := builders[i]; ok {
			toolCalls = append(toolCalls, domain.ToolCall{ID: b.id, Name: b.name, Arguments: b.arguments})
		}
	}
	events <- domain.StreamEvent{Done: true, ToolCalls: toolCalls, Usage: usage}
}

func toWireMessages(messages []domain.Message) []wireMessage {
	out := make([]wireMessage, len(messages))
	for i, m := range messages {
		wm := wireMessage{
			Role:       string(m.Role),
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		for _, tc := range m.ToolCalls {
			wtc := wireToolCall{ID: tc.ID, Type: "function"}
			wtc.Function.Name = tc.Name
			wtc.Function.Arguments = tc.Arguments
			wm.ToolCalls = append(wm.ToolCalls, wtc)
		}
		out[i] = wm
	}
	return out
}

func toWireTools(tools []domain.ToolDefinition) []wireTool {
	if len(tools) == 0 {
		return nil
	}
	out := make([]wireTool, len(tools))
	for i, t := range tools {
		wt := wireTool{Type: "function"}
		wt.Function.Name = t.Name
		wt.Function.Description = t.Description
		wt.Function.Parameters = t.Parameters
		out[i] = wt
	}
	return out
}

func fromWireMessage(m wireMessage) domain.Message {
	dm := domain.Message{
		Role:       domain.Role(m.Role),
		Content:    m.Content,
		ToolCallID: m.ToolCallID,
	}
	for _, wtc := range m.ToolCalls {
		dm.ToolCalls = append(dm.ToolCalls, domain.ToolCall{
			ID:        wtc.ID,
			Name:      wtc.Function.Name,
			Arguments: wtc.Function.Arguments,
		})
	}
	return dm
}
