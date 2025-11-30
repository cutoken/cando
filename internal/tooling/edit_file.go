package tooling

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

// EditFileTool performs exact string replacements in files.
type EditFileTool struct {
	guard pathGuard
}

func NewEditFileTool(guard pathGuard) *EditFileTool {
	return &EditFileTool{guard: guard}
}

func (EditFileTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "edit_file",
			Description: "Perform exact string replacement in a file. The old_string must match exactly (including whitespace and indentation). Use read_file first to see the current content. If this fails with 'old_string not found', re-read the file before retrying - the content may have changed. This is safer than write_file for making targeted changes.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to the file to edit, relative to workspace root.",
					},
					"old_string": map[string]any{
						"type":        "string",
						"description": "The exact string to replace (must match exactly including whitespace).",
					},
					"new_string": map[string]any{
						"type":        "string",
						"description": "The replacement string.",
					},
					"replace_all": map[string]any{
						"type":        "boolean",
						"description": "If true, replace all occurrences. If false (default), old_string must be unique in the file.",
					},
				},
				"required": []string{"path", "old_string", "new_string"},
			},
		},
	}
}

func (e *EditFileTool) Call(ctx context.Context, args map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	path, ok := stringArg(args, "path")
	if !ok || path == "" {
		return "", errors.New("path is required")
	}

	oldString, ok := stringArg(args, "old_string")
	if !ok {
		return "", errors.New("old_string is required")
	}

	newString, ok := stringArg(args, "new_string")
	if !ok {
		return "", errors.New("new_string is required")
	}

	if oldString == newString {
		return "", errors.New("old_string and new_string must be different")
	}

	replaceAll := boolArg(args, "replace_all", false)

	absPath, err := e.guard.Resolve(path)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	contentStr := string(content)
	count := strings.Count(contentStr, oldString)
	if count == 0 {
		snippet := oldString
		const maxPreview = 80
		if len(snippet) > maxPreview {
			snippet = snippet[:maxPreview] + "…"
		}
		return "", fmt.Errorf("old_string not found. Double-check whitespace/indentation. Preview: %q", snippet)
	}

	if !replaceAll && count > 1 {
		return "", fmt.Errorf("old_string appears %d times in the file. Use replace_all=true to replace all occurrences, or provide a larger unique string", count)
	}

	var newContent string
	if replaceAll {
		newContent = strings.ReplaceAll(contentStr, oldString, newString)
	} else {
		newContent = strings.Replace(contentStr, oldString, newString, 1)
	}

	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	replacedCount := count
	if !replaceAll {
		replacedCount = 1
	}

	return fmt.Sprintf("Successfully replaced %d occurrence(s) in %s", replacedCount, path), nil
}
