// Package domain defines the AI chat module's core types: the OpenAI-
// compatible chat-completion shapes (Message/ToolCall/ToolDefinition) and
// the LLMClient port. Kept provider-agnostic on purpose — DeepSeek is the
// first implementation (internal/ai/infrastructure), but every cheap LLM
// provider worth adding later (GLM, Moonshot, Qwen, ...) speaks this same
// OpenAI-compatible chat-completions shape, so a new provider is a new
// infrastructure constructor pointed at a different base URL/model, not a
// change to this package.
package domain

import (
	"context"
	"encoding/json"
)

// Role is who a Message is from, using the OpenAI chat-completions
// vocabulary every OpenAI-compatible provider (DeepSeek, GLM, ...) shares.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	// RoleTool marks a message carrying a tool's result, sent back to the
	// model after it requested a ToolCall — see Message.ToolCallID.
	RoleTool Role = "tool"
)

// ToolCall is one function invocation the model asked the caller to run,
// attached to an assistant Message.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string // raw JSON object, as the model produced it
}

// Message is one turn in the conversation sent to/received from the model.
type Message struct {
	Role    Role
	Content string
	// ToolCalls is set on an assistant Message that wants tool(s) run
	// instead of (or before) answering.
	ToolCalls []ToolCall
	// ToolCallID is set on a RoleTool message — which ToolCall this is the
	// result of.
	ToolCallID string
}

// ToolDefinition describes a callable tool to the model, using the OpenAI
// function-calling schema (Parameters is a JSON Schema object).
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

// ChatResult is what one round-trip to the model returns: either a final
// answer (Message.Content, no ToolCalls) or a request to run tool(s)
// (Message.ToolCalls non-empty) — the caller decides which by checking
// ToolCalls, same as the OpenAI API convention this mirrors.
type ChatResult struct {
	Message Message
	// Usage is the token counts this call billed against, as reported by
	// the provider — zero value if the provider omitted it. Used by
	// SendChatMessage to compute Pricing.Cost after the fallback loop
	// settles on a model.
	Usage TokenUsage
}

// LLMClient is a chat-completion round trip against an OpenAI-compatible
// provider. One call = one model turn; the tool-call loop (call, run tools,
// call again) is orchestrated by internal/ai/application, not by this port.
type LLMClient interface {
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*ChatResult, error)
}

// LLMClientFactory builds an LLMClient for one resolved ModelConfig. The
// fallback loop (application.SendChatMessage/ClassifyMessage) needs a fresh
// client per attempt — each ModelConfig carries a different base URL/API
// key/model — so this is injected as a factory rather than a single client
// instance, keeping application decoupled from the concrete infrastructure
// constructor (wired once at the composition root, cmd/server/main.go).
type LLMClientFactory func(cfg ModelConfig) LLMClient
