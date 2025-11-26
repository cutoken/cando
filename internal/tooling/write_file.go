package tooling

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteFileTool edits files within the workspace.
type WriteFileTool struct {
	guard pathGuard
}

func NewWriteFileTool(guard pathGuard) *WriteFileTool {
	return &WriteFileTool{guard: guard}
}

func (t *WriteFileTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "write_file",
			Description: "Write text to a file. Supports append, inserting at a specific line, or replacing a line range.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to the file relative to the workspace root.",
					},
					"mode": map[string]any{
						"type":        "string",
						"description": "append (default), insert, or replace.",
					},
					"line": map[string]any{
						"type":        "integer",
						"description": "For insert mode: the 1-based line number to insert before (defaults to end).",
					},
					"start_line": map[string]any{
						"type":        "integer",
						"description": "For replace mode: starting line number (1-based).",
					},
					"end_line": map[string]any{
						"type":        "integer",
						"description": "For replace mode: ending line number (inclusive).",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Text to write. Use \n for new lines.",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}

func (t *WriteFileTool) Call(ctx context.Context, args map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	path, ok := stringArg(args, "path")
	if !ok || strings.TrimSpace(path) == "" {
		return "", errors.New("path is required")
	}
	abs, err := t.guard.Resolve(path)
	if err != nil {
		return "", err
	}

	content, ok := stringArg(args, "content")
	if !ok {
		return "", errors.New("content is required")
	}

	mode, _ := stringArg(args, "mode")
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "append"
	}

	switch mode {
	case "append":
		return t.append(abs, content)
	case "insert":
		line := intArg(args, "line", -1)
		if line == 0 {
			return "", errors.New("line numbers are 1-based")
		}
		return t.insert(abs, line, content)
	case "replace":
		start := intArg(args, "start_line", 0)
		end := intArg(args, "end_line", 0)
		if start <= 0 || end <= 0 {
			return "", errors.New("start_line and end_line must be positive for replace")
		}
		if end < start {
			start, end = end, start
		}
		return t.replaceRange(abs, start, end, content)
	default:
		return "", fmt.Errorf("unsupported mode %s", mode)
	}
}

func (t *WriteFileTool) append(abs string, content string) (string, error) {
	f, err := os.OpenFile(abs, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return "", err
	}
	payload := map[string]any{"path": t.guard.Rel(abs), "mode": "append", "bytes": len(content)}
	data, _ := json.Marshal(payload)
	return string(data), nil
}

func (t *WriteFileTool) insert(abs string, line int, content string) (string, error) {
	lines, trailing, err := readLines(abs)
	if err != nil {
		return "", err
	}
	insertAt := len(lines)
	if line > 0 && line-1 < len(lines) {
		insertAt = line - 1
	}
	if line <= 1 {
		insertAt = 0
	}
	newLines := splitContent(content)
	updated := append(lines[:insertAt], append(newLines, lines[insertAt:]...)...)
	if err := writeLines(abs, updated, trailing); err != nil {
		return "", err
	}
	payload := map[string]any{
		"path":       t.guard.Rel(abs),
		"mode":       "insert",
		"line":       insertAt + 1,
		"linesAdded": len(newLines),
	}
	data, _ := json.Marshal(payload)
	return string(data), nil
}

func (t *WriteFileTool) replaceRange(abs string, start, end int, content string) (string, error) {
	lines, trailing, err := readLines(abs)
	if err != nil {
		return "", err
	}
	if start > len(lines) {
		start = len(lines) + 1
	}
	if start <= 0 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	startIdx := start - 1
	endIdx := end
	if endIdx < startIdx {
		endIdx = startIdx
	}
	if endIdx > len(lines) {
		endIdx = len(lines)
	}
	newLines := splitContent(content)
	updated := append(append([]string{}, lines[:startIdx]...), append(newLines, lines[endIdx:]...)...)
	if err := writeLines(abs, updated, trailing); err != nil {
		return "", err
	}
	payload := map[string]any{
		"path":         t.guard.Rel(abs),
		"mode":         "replace",
		"start_line":   start,
		"end_line":     end,
		"linesWritten": len(newLines),
	}
	data, _ := json.Marshal(payload)
	return string(data), nil
}

func readLines(path string) ([]string, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	trailing := len(data) > 0 && data[len(data)-1] == '\n'
	text := strings.TrimRight(string(data), "\n")
	if text == "" {
		if len(data) == 0 {
			return []string{}, false, nil
		}
		return []string{}, true, nil
	}
	return strings.Split(text, "\n"), trailing, nil
}

func writeLines(path string, lines []string, trailing bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	if trailing {
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func splitContent(content string) []string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.Split(normalized, "\n")
}
