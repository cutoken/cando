package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
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
		return respPayload, parseOpenRouterError(resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, &respPayload); err != nil {
		logging.ErrorLog("openrouter response parse error: %v", err)
		return respPayload, fmt.Errorf("parse response: %w", err)
	}
	logging.DevLog("openrouter: received response with %d choices", len(respPayload.Choices))
	return respPayload, nil
}

// parseOpenRouterError converts OpenRouter error responses to structured ProviderError
func parseOpenRouterError(statusCode int, body []byte) *llm.ProviderError {
	pe := &llm.ProviderError{
		Provider: "openrouter",
		Code:     strconv.Itoa(statusCode),
	}

	// Try to parse structured error response
	// Format: {"error":{"code":429,"message":"...","metadata":{"headers":{...}}}}
	var errResp struct {
		Error struct {
			Code     int    `json:"code"`
			Message  string `json:"message"`
			Metadata struct {
				Headers struct {
					RateLimitLimit     string `json:"X-RateLimit-Limit"`
					RateLimitRemaining string `json:"X-RateLimit-Remaining"`
					RateLimitReset     string `json:"X-RateLimit-Reset"`
				} `json:"headers"`
			} `json:"metadata"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		pe.Message = errResp.Error.Message
		if errResp.Error.Code != 0 {
			pe.Code = strconv.Itoa(errResp.Error.Code)
		}

		// Parse reset time from metadata (Unix timestamp in milliseconds)
		if resetMs := errResp.Error.Metadata.Headers.RateLimitReset; resetMs != "" {
			if ms, err := strconv.ParseInt(resetMs, 10, 64); err == nil {
				t := time.UnixMilli(ms)
				pe.ResetAt = &t
				// Calculate retry duration
				wait := time.Until(t)
				if wait > 0 {
					pe.RetryAfter = &wait
				}
			}
		}
	} else {
		// Couldn't parse JSON, use raw body
		pe.Message = strings.TrimSpace(string(body))
	}

	// Classify error type based on status code
	switch statusCode {
	case 402:
		pe.Type = llm.ErrorTypeInsufficientCredit
		pe.Retryable = false
		if pe.Message == "" {
			pe.Message = "Insufficient credits. Please add balance to your OpenRouter account."
		}
	case 429:
		pe.Type = llm.ErrorTypeRateLimit
		// Only auto-retry if reset is reasonably soon (< 5 minutes)
		if pe.RetryAfter != nil && *pe.RetryAfter < 5*time.Minute {
			pe.Retryable = true
		} else if pe.RetryAfter != nil {
			// Long wait (daily limit) - don't auto-retry
			pe.Retryable = false
		} else {
			// No reset time known - try with backoff
			pe.Retryable = true
			defaultDelay := 30 * time.Second
			pe.RetryAfter = &defaultDelay
		}
	case 401:
		pe.Type = llm.ErrorTypeAuth
		pe.Retryable = false
		if pe.Message == "" {
			pe.Message = "Invalid API key. Please check your OpenRouter credentials."
		}
	case 403:
		pe.Type = llm.ErrorTypeModeration
		pe.Retryable = false
	case 502, 503:
		pe.Type = llm.ErrorTypeProviderDown
		pe.Retryable = true
		defaultDelay := 10 * time.Second
		pe.RetryAfter = &defaultDelay
	default:
		pe.Type = llm.ErrorTypeUnknown
		pe.Retryable = statusCode >= 500
	}

	return pe
}
