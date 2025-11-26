package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// GlobTool finds files matching glob patterns.
type GlobTool struct {
	guard pathGuard
}

func NewGlobTool(guard pathGuard) *GlobTool {
	return &GlobTool{guard: guard}
}

func (GlobTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "glob",
			Description: "Find files matching a glob pattern like '**/*.go' or 'src/**/*.ts'. Returns matching file paths sorted by modification time (most recent first). Use this for pattern-based file searches.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{
						"type":        "string",
						"description": "Glob pattern to match files. Examples: '*.js', '**/*.go', 'src/**/*.ts'",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "Directory to search in (default: workspace root).",
					},
					"max_results": map[string]any{
						"type":        "integer",
						"description": "Maximum number of results to return (default: 100).",
					},
				},
				"required": []string{"pattern"},
			},
		},
	}
}

func (g *GlobTool) Call(ctx context.Context, args map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	pattern, ok := stringArg(args, "pattern")
	if !ok || pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	searchPath := ""
	if provided, ok := stringArg(args, "path"); ok {
		searchPath = provided
	}

	root, err := g.guard.Resolve(searchPath)
	if err != nil {
		return "", err
	}

	maxResults := intArg(args, "max_results", 100)
	if maxResults <= 0 {
		maxResults = 100
	}

	fullPattern := filepath.Join(root, pattern)

	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return "", fmt.Errorf("glob pattern error: %w", err)
	}

	type fileInfo struct {
		Path    string
		ModTime time.Time
	}

	var files []fileInfo
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue // Skip files we can't stat
		}

		// Skip directories
		if info.IsDir() {
			continue
		}

		relPath, err := filepath.Rel(g.guard.root, match)
		if err != nil {
			relPath = match
		}

		files = append(files, fileInfo{
			Path:    relPath,
			ModTime: info.ModTime(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})

	if len(files) > maxResults {
		files = files[:maxResults]
	}

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Path
	}

	result := map[string]any{
		"pattern": pattern,
		"count":   len(paths),
		"files":   paths,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
