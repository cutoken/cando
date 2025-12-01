package tooling

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PreviewStateKey is used to pass preview enabled state through context
type previewStateKey struct{}

// WithPreviewState adds preview enabled state to context
func WithPreviewState(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, previewStateKey{}, enabled)
}

// PreviewStateFromContext gets preview enabled state from context (default true)
func PreviewStateFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(previewStateKey{}).(bool); ok {
		return v
	}
	return true // Default enabled
}

// PreviewFileTool shows files in the preview pane.
type PreviewFileTool struct {
	guard pathGuard
}

func NewPreviewFileTool(guard pathGuard) *PreviewFileTool {
	return &PreviewFileTool{guard: guard}
}

func (PreviewFileTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "preview_file",
			Description: "Display a file in the preview pane. Use this to show HTML pages, images, PDFs, or other files to the user. The preview pane appears in the UI allowing the user to see the result without manually opening files.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to the file to preview (relative to workspace root).",
					},
					"title": map[string]any{
						"type":        "string",
						"description": "Optional title for the preview pane.",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (p *PreviewFileTool) Call(ctx context.Context, args map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	path, ok := stringArg(args, "path")
	if !ok || strings.TrimSpace(path) == "" {
		return "", errors.New("path is required")
	}

	title, _ := stringArg(args, "title")
	if title == "" {
		title = filepath.Base(path)
	}

	// Resolve and validate path
	absPath, err := p.guard.Resolve(path)
	if err != nil {
		return "", err
	}

	// Check file exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", path)
		}
		return "", fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("cannot preview directories: %s", path)
	}

	// Get relative path for result
	relPath := p.guard.Rel(absPath)

	// Check preview state from context
	previewEnabled := PreviewStateFromContext(ctx)

	// Build result based on preview state
	result := map[string]any{
		"path":            relPath,
		"title":           title,
		"preview_enabled": previewEnabled,
	}

	if previewEnabled {
		result["message"] = fmt.Sprintf("Preview displayed: %s", relPath)
		result["action"] = "preview_shown"
	} else {
		result["message"] = fmt.Sprintf("Preview pane is disabled by user. The file is available at: %s. Ask the user to enable the preview pane or open the file manually.", relPath)
		result["action"] = "preview_disabled"
	}

	data, err := jsonMarshalNoEscape(result)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// SupportedPreviewTypes returns file extensions that can be previewed
func SupportedPreviewTypes() []string {
	return []string{
		".html", ".htm",
		".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico",
		".pdf",
		".txt", ".md", ".json", ".xml", ".csv",
	}
}

// CanPreview checks if a file type is supported for preview
func CanPreview(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, supported := range SupportedPreviewTypes() {
		if ext == supported {
			return true
		}
	}
	return false
}
