// Package infrastructure implements internal/ai/domain's ports: the
// LLMClient against an OpenAI-compatible chat-completions API, and the
// product-search tool the model can call.
package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

type wireRequest struct {
	Model    string        `json:"model"`
	Messages []wireMessage `json:"messages"`
	Tools    []wireTool    `json:"tools,omitempty"`
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
