package zai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"cando/internal/llm"
	"cando/internal/state"
)

// No hardcoded endpoint - must come from config

// ZAIResponse represents the full response structure from Z.AI API.
type ZAIResponse struct {
	Choices []ZAIChoice `json:"choices"`
	Usage   *llm.Usage  `json:"usage,omitempty"`
}

// ZAIChoice represents a single choice in Z.AI response.
type ZAIChoice struct {
	Index        int        `json:"index"`
	Message      ZAIMessage `json:"message"`
	FinishReason string     `json:"finish_reason"`
}

// ZAIMessage represents message content from Z.AI.
type ZAIMessage struct {
	Content          string           `json:"content,omitempty"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	Thinking         string           `json:"thinking,omitempty"`
	Reasoning        []ReasoningStep  `json:"reasoning,omitempty"`
	ContentBlocks    []ContentBlock   `json:"content_blocks,omitempty"`
	ToolCalls        []state.ToolCall `json:"tool_calls,omitempty"`
}

// ReasoningStep represents a single step in Z.AI reasoning process.
type ReasoningStep struct {
	Step        string `json:"step"`
	Explanation string `json:"explanation"`
}

// ContentBlock represents a structured content block in Z.AI responses.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Client wraps Z.AI chat completion API.
type Client struct {
	httpClient     *http.Client
	endpoint       string
	apiKey         string
	logger         *log.Logger
	acceptLanguage string
}

// NewClient configures a Z.AI completion client.
func NewClient(endpoint, apiKey string, timeout time.Duration, logger *log.Logger) *Client {
	trimmed := strings.TrimRight(endpoint, "/")
	if trimmed == "" {
		panic("ZAI endpoint must be provided from config - no hardcoded defaults")
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Client{
		httpClient:     &http.Client{Timeout: timeout},
		endpoint:       trimmed,
		apiKey:         apiKey,
		logger:         logger,
		acceptLanguage: "en-US,en",
	}
}

// parseZAIResponse extracts thinking content from Z.AI responses.
func (c *Client) parseZAIResponse(zaiResp *ZAIResponse) (llm.ChatResponse, error) {
	if len(zaiResp.Choices) == 0 {
		return llm.ChatResponse{}, fmt.Errorf("no choices in response")
	}

	zaiChoice := zaiResp.Choices[0]
	var mainContent, thinkingContent strings.Builder

	// Handle different Z.AI response formats
	if zaiChoice.Message.ReasoningContent != "" {
		// Primary Z.AI thinking field (documented format)
		thinkingContent.WriteString(zaiChoice.Message.ReasoningContent)
		mainContent.WriteString(zaiChoice.Message.Content)
	} else if zaiChoice.Message.Thinking != "" {
		// Alternative thinking field
		thinkingContent.WriteString(zaiChoice.Message.Thinking)
		mainContent.WriteString(zaiChoice.Message.Content)
	} else if len(zaiChoice.Message.ContentBlocks) > 0 {
		// Structured content blocks
		for _, block := range zaiChoice.Message.ContentBlocks {
			if block.Type == "thinking" {
				thinkingContent.WriteString(block.Text)
				thinkingContent.WriteString("\n\n")
			} else if block.Type == "text" {
				mainContent.WriteString(block.Text)
			}
		}
	} else if len(zaiChoice.Message.Reasoning) > 0 {
		// Reasoning steps format
		for _, step := range zaiChoice.Message.Reasoning {
			thinkingContent.WriteString(fmt.Sprintf("Step: %s\n%s\n\n", step.Step, step.Explanation))
		}
		mainContent.WriteString(zaiChoice.Message.Content)
	} else {
		// Fallback to regular content
		mainContent.WriteString(zaiChoice.Message.Content)
	}

	// Update message with thinking content
	if thinkingContent.Len() > 0 {
		zaiChoice.Message.Thinking = thinkingContent.String()
	}

	// If content is empty but we have thinking, use thinking as content
	// This handles cases where Z.AI only returns reasoning_content with no regular content
	if strings.TrimSpace(mainContent.String()) == "" && thinkingContent.Len() > 0 {
		zaiChoice.Message.Content = thinkingContent.String()
	} else {
		zaiChoice.Message.Content = mainContent.String()
	}

	return llm.ChatResponse{
		Choices: []llm.ChatChoice{
			{
				Index:        zaiChoice.Index,
				Message:      convertZAIMessageToStandard(zaiChoice.Message),
				FinishReason: zaiChoice.FinishReason,
			},
		},
		Usage: zaiResp.Usage,
	}, nil
}

// convertZAIMessageToStandard converts Z.AI message to standard llm.Message format.
func convertZAIMessageToStandard(zaiMsg ZAIMessage) state.Message {
	return state.Message{
		Role:      "assistant",
		Content:   zaiMsg.Content,
		Thinking:  zaiMsg.Thinking,
		ToolCalls: zaiMsg.ToolCalls,
	}
}

// Chat satisfies the llm.Client interface.
func (c *Client) Chat(ctx context.Context, reqPayload llm.ChatRequest) (llm.ChatResponse, error) {
	var respPayload llm.ChatResponse

	body, err := json.Marshal(reqPayload)
	if err != nil {
		return respPayload, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return respPayload, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if c.acceptLanguage != "" {
		req.Header.Set("Accept-Language", c.acceptLanguage)
	}

	c.logger.Printf("[z.ai] sending %d messages to model %s", len(reqPayload.Messages), reqPayload.Model)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return respPayload, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return respPayload, fmt.Errorf("read response: %w", err)
	}

	// Log response status
	c.logger.Printf("[z.ai] Response status: %d, size: %d bytes", resp.StatusCode, len(respBody))

	// Check for Z.AI custom error format (returns 200 with error object)
	// Only treat as error if it has the error structure (code + msg fields)
	type errorResponse struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		Success bool   `json:"success"`
	}
	var errResp errorResponse
	if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Code != 0 && errResp.Msg != "" {
		c.logger.Printf("[z.ai] API returned error: code=%d msg=%s", errResp.Code, errResp.Msg)
		return respPayload, fmt.Errorf("z.ai api error: %s", errResp.Msg)
	}

	if resp.StatusCode >= 300 {
		return respPayload, fmt.Errorf("api error: %s", strings.TrimSpace(string(respBody)))
	}

	// Try to parse as Z.AI enhanced response first
	var zaiResp ZAIResponse
	if err := json.Unmarshal(respBody, &zaiResp); err == nil {
		if len(zaiResp.Choices) == 0 {
			return llm.ChatResponse{}, fmt.Errorf("no choices returned")
		}
		return c.parseZAIResponse(&zaiResp)
	}

	// Fallback to standard parsing
	if err := json.Unmarshal(respBody, &respPayload); err != nil {
		return respPayload, fmt.Errorf("parse response: %w", err)
	}
	if len(respPayload.Choices) == 0 {
		return respPayload, fmt.Errorf("no choices returned")
	}
	return respPayload, nil
}
