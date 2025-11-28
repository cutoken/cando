package openrouter

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
	"cando/internal/logging"
)

// Client is a minimal HTTP wrapper around the OpenRouter chat completions API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	logger     *log.Logger
}

// NewClient wires together the dependencies for API access.
func NewClient(baseURL, apiKey string, timeout time.Duration, logger *log.Logger) *Client {
	trimmed := strings.TrimRight(baseURL, "/")
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    trimmed,
		apiKey:     apiKey,
		logger:     logger,
	}
}

// Chat executes a single completion request.
func (c *Client) Chat(ctx context.Context, reqPayload llm.ChatRequest) (llm.ChatResponse, error) {
	var respPayload llm.ChatResponse

	payload, err := json.Marshal(reqPayload)
	if err != nil {
		return respPayload, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := c.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return respPayload, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/cutoken/cando")
	req.Header.Set("X-Title", "Cando")

	c.logger.Printf("sending %d messages to model %s", len(reqPayload.Messages), reqPayload.Model)
	logging.DevLog("openrouter: sending request to %s with %d messages", reqPayload.Model, len(reqPayload.Messages))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return respPayload, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return respPayload, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		logging.ErrorLog("openrouter API error: %d - %s", resp.StatusCode, string(body))
		return respPayload, fmt.Errorf("api error: %s", string(body))
	}

	if err := json.Unmarshal(body, &respPayload); err != nil {
		logging.ErrorLog("openrouter response parse error: %v", err)
		return respPayload, fmt.Errorf("parse response: %w", err)
	}
	logging.DevLog("openrouter: received response with %d choices", len(respPayload.Choices))
	return respPayload, nil
}
