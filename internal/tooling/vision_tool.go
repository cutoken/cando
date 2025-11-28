package tooling

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxImageSize = 5 * 1024 * 1024 // 5MB - Z.AI's limit
)

// VisionTool analyzes images using provider-specific vision APIs.
type VisionTool struct {
	guard       pathGuard
	credManager CredentialManager
	client      *http.Client
	config      VisionConfig
}

// VisionConfig holds endpoints from the main config
type VisionConfig struct {
	ZAIEndpoint        string
	OpenRouterEndpoint string
}

// NewVisionTool constructs a vision analysis tool.
func NewVisionTool(guard pathGuard, credManager CredentialManager) *VisionTool {
	return &VisionTool{
		guard:       guard,
		credManager: credManager,
		client:      &http.Client{Timeout: 60 * time.Second},
	}
}

// NewVisionToolWithConfig constructs a vision tool with config endpoints.
func NewVisionToolWithConfig(guard pathGuard, credManager CredentialManager, zaiEndpoint, openrouterEndpoint string) *VisionTool {
	return &VisionTool{
		guard:       guard,
		credManager: credManager,
		client:      &http.Client{Timeout: 60 * time.Second},
		config: VisionConfig{
			ZAIEndpoint:        zaiEndpoint,
			OpenRouterEndpoint: openrouterEndpoint,
		},
	}
}

func (v *VisionTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "analyze_image",
			Description: "Analyze an image file using vision AI to answer questions about its contents, extract information, or describe what it shows.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"image_path": map[string]any{
						"type":        "string",
						"description": "Path to the image file (relative to workspace root).",
					},
					"prompt": map[string]any{
						"type":        "string",
						"description": "Question or instruction about what to analyze in the image.",
					},
				},
				"required": []string{"image_path", "prompt"},
			},
		},
	}
}

func (v *VisionTool) Call(ctx context.Context, args map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Extract arguments
	imagePath, ok := stringArg(args, "image_path")
	if !ok || imagePath == "" {
		return "", errors.New("image_path is required")
	}

	prompt, ok := stringArg(args, "prompt")
	if !ok || prompt == "" {
		return "", errors.New("prompt is required")
	}

	// Load credentials to check provider configuration
	if v.credManager == nil {
		return "", errors.New("vision tool requires credential manager")
	}

	creds, err := v.credManager.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load credentials: %w", err)
	}

	// Determine which provider to use (prefer ZAI, fall back to OpenRouter)
	var provider, apiKey, visionModel string

	if creds.IsConfigured("zai") {
		provider = "zai"
		apiKey = creds.GetAPIKey("zai")
		visionModel = creds.GetVisionModel("zai")
		if visionModel == "" {
			return "", errors.New("vision model not configured for Z.AI")
		}
	} else if creds.IsConfigured("openrouter") {
		provider = "openrouter"
		apiKey = creds.GetAPIKey("openrouter")
		visionModel = creds.GetVisionModel("openrouter")
		if visionModel == "" {
			return "", errors.New("vision model not configured for OpenRouter")
		}
	} else {
		return "", errors.New("vision analysis requires Z.AI or OpenRouter provider - configure API key in settings")
	}

	if apiKey == "" {
		return "", fmt.Errorf("%s API key not found", provider)
	}

	// Resolve and validate image path within workspace
	absPath, err := v.guard.Resolve(imagePath)
	if err != nil {
		return "", fmt.Errorf("invalid image path: %w", err)
	}

	// Check file size before reading
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to access image: %w", err)
	}

	if info.Size() > maxImageSize {
		return "", fmt.Errorf("image size (%d bytes) exceeds 5MB limit", info.Size())
	}

	// Read and encode image
	imageData, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}

	// Detect MIME type from file extension
	mimeType := "image/jpeg" // default
	ext := strings.ToLower(filepath.Ext(absPath))
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	}

	// Encode to base64 and build data URI
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	// Call appropriate vision API
	var result string
	switch provider {
	case "zai":
		result, err = v.callZAIVision(ctx, apiKey, visionModel, dataURI, prompt)
	case "openrouter":
		result, err = v.callOpenRouterVision(ctx, apiKey, visionModel, dataURI, prompt)
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return "", err
	}

	// Return JSON response
	relPath := v.guard.Rel(absPath)
	payload := map[string]any{
		"image_path": relPath,
		"prompt":     prompt,
		"provider":   provider,
		"model":      visionModel,
		"analysis":   result,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// callZAIVision makes the HTTP request to Z.AI's vision endpoint.
func (v *VisionTool) callZAIVision(ctx context.Context, apiKey, model, imageDataURI, prompt string) (string, error) {
	type imageURLDetail struct {
		URL string `json:"url"`
	}

	type imageContent struct {
		Type     string          `json:"type"`
		ImageURL *imageURLDetail `json:"image_url,omitempty"`
		Text     string          `json:"text,omitempty"`
	}

	type message struct {
		Role    string         `json:"role"`
		Content []imageContent `json:"content"`
	}

	type visionRequest struct {
		Model    string    `json:"model"`
		Messages []message `json:"messages"`
	}

	req := visionRequest{
		Model: model,
		Messages: []message{
			{
				Role: "user",
				Content: []imageContent{
					{
						Type:     "image_url",
						ImageURL: &imageURLDetail{URL: imageDataURI},
					},
					{
						Type: "text",
						Text: prompt,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	if v.config.ZAIEndpoint == "" {
		return "", fmt.Errorf("ZAI vision endpoint not configured")
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", v.config.ZAIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept-Language", "en-US,en")

	resp, err := v.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("vision API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	type visionResponse struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content,omitempty"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	var result visionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("vision API error %s: %s", result.Error.Code, result.Error.Message)
	}

	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		return "", errors.New("vision API returned empty response")
	}

	return result.Choices[0].Message.Content, nil
}

// callOpenRouterVision makes the HTTP request to OpenRouter's vision endpoint.
func (v *VisionTool) callOpenRouterVision(ctx context.Context, apiKey, model, imageDataURI, prompt string) (string, error) {
	type imageURLDetail struct {
		URL string `json:"url"`
	}

	type contentPart struct {
		Type     string          `json:"type"`
		Text     string          `json:"text,omitempty"`
		ImageURL *imageURLDetail `json:"image_url,omitempty"`
	}

	type message struct {
		Role    string        `json:"role"`
		Content []contentPart `json:"content"`
	}

	type visionRequest struct {
		Model    string    `json:"model"`
		Messages []message `json:"messages"`
	}

	req := visionRequest{
		Model: model,
		Messages: []message{
			{
				Role: "user",
				Content: []contentPart{
					{
						Type:     "image_url",
						ImageURL: &imageURLDetail{URL: imageDataURI},
					},
					{
						Type: "text",
						Text: prompt,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	if v.config.OpenRouterEndpoint == "" {
		return "", fmt.Errorf("OpenRouter vision endpoint not configured")
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", v.config.OpenRouterEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := v.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("vision API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response (OpenAI-compatible format)
	type visionResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error,omitempty"`
	}

	var result visionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("OpenRouter API error %s: %s", result.Error.Code, result.Error.Message)
	}

	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		return "", errors.New("OpenRouter API returned empty response")
	}

	return result.Choices[0].Message.Content, nil
}
