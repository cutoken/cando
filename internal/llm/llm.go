package llm

import (
	"context"

	"cando/internal/state"
	"cando/internal/tooling"
)

// ChatRequest is the provider-agnostic message payload for chat completions.
type ChatRequest struct {
	Model       string                   `json:"model"`
	Messages    []state.Message          `json:"messages"`
	Tools       []tooling.ToolDefinition `json:"tools,omitempty"`
	Temperature float64                  `json:"temperature,omitempty"`
	Thinking    *ThinkingOptions         `json:"thinking,omitempty"`
}

type ThinkingOptions struct {
	Type         string `json:"type"`               // "enabled" or "disabled" for Z.AI
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

// ChatChoice captures one response alternative from a completion API.
type ChatChoice struct {
	Index        int           `json:"index"`
	Message      state.Message `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// Usage contains token consumption metrics from the LLM API.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse is the shared representation of provider responses.
type ChatResponse struct {
	Choices []ChatChoice `json:"choices"`
	Usage   *Usage       `json:"usage,omitempty"`
}

// Client represents an LLM provider capable of servicing chat completions.
type Client interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
