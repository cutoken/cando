package mockclient

import (
	"context"
	"fmt"
	"strings"

	"cando/internal/llm"
	"cando/internal/state"
)

// Client is a deterministic llm.Client used for tests and CI.
type Client struct {
	prefix string
}

// New returns a mock client that echoes the last user message.
func New() *Client {
	return &Client{prefix: "MOCK"}
}

// Chat satisfies the llm.Client interface.
func (c *Client) Chat(_ context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	response := state.Message{
		Role: "assistant",
	}

	if n := len(req.Messages); n > 0 {
		last := req.Messages[n-1].Content
		last = strings.TrimSpace(last)
		if last == "" {
			response.Content = fmt.Sprintf("%s RESPONSE", c.prefix)
		} else {
			response.Content = fmt.Sprintf("%s RESPONSE: %s", c.prefix, last)
		}
	} else {
		response.Content = fmt.Sprintf("%s RESPONSE", c.prefix)
	}

	return llm.ChatResponse{
		Choices: []llm.ChatChoice{
			{
				Index:        0,
				Message:      response,
				FinishReason: "stop",
			},
		},
		Usage: &llm.Usage{
			PromptTokens:     42,
			CompletionTokens: 7,
			TotalTokens:      49,
		},
	}, nil
}
