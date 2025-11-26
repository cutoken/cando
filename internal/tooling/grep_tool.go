package tooling

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepTool searches file contents using regex patterns.
type GrepTool struct {
	guard pathGuard
}

// grepLine represents a single line in grep output with separate line number and content
type grepLine struct {
	Line    int    `json:"line"`
	Type    string `json:"type"` // "match" or "context"
	Content string `json:"content"`
}

func NewGrepTool(guard pathGuard) *GrepTool {
	return &GrepTool{guard: guard}
}

func (GrepTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "grep",
			Description: "Search file contents using regex patterns. Returns matching lines or file paths. In content mode, each match is an array of line objects with separate 'line' (number), 'type' (match/context), and 'content' (exact text) fields. Supports context lines, case-insensitive search, and file type filtering.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{
						"type":        "string",
						"description": "Regular expression pattern to search for.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "File or directory to search (default: workspace root).",
					},
					"glob": map[string]any{
						"type":        "string",
						"description": "Glob pattern to filter files (e.g., '*.js', '*.{ts,tsx}').",
					},
					"case_insensitive": map[string]any{
						"type":        "boolean",
						"description": "Perform case-insensitive search (default: false).",
					},
					"output_mode": map[string]any{
						"type":        "string",
						"description": "Output mode: 'content' (show matching lines), 'files' (show file paths only), 'count' (show match counts). Default: 'files'.",
						"enum":        []string{"content", "files", "count"},
					},
					"context_before": map[string]any{
						"type":        "integer",
						"description": "Number of lines to show before each match (requires output_mode='content').",
					},
					"context_after": map[string]any{
						"type":        "integer",
						"description": "Number of lines to show after each match (requires output_mode='content').",
					},
					"max_results": map[string]any{
						"type":        "integer",
						"description": "Maximum number of results to return (default: 100).",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Skip first N matches (for pagination when results are truncated). Default: 0.",
					},
				},
				"required": []string{"pattern"},
			},
		},
	}
}

func (g *GrepTool) Call(ctx context.Context, args map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	patternStr, ok := stringArg(args, "pattern")
	if !ok || patternStr == "" {
		return "", errors.New("pattern is required")
	}

	caseInsensitive := boolArg(args, "case_insensitive", false)
	if caseInsensitive {
		patternStr = "(?i)" + patternStr
	}

	pattern, err := regexp.Compile(patternStr)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	searchPath := ""
	if provided, ok := stringArg(args, "path"); ok {
		searchPath = provided
	}

	root, err := g.guard.Resolve(searchPath)
	if err != nil {
		return "", err
	}

	globPattern, _ := stringArg(args, "glob")
	outputMode, _ := stringArg(args, "output_mode")
	if outputMode == "" {
		outputMode = "files"
	}

	contextBefore := intArg(args, "context_before", 0)
	contextAfter := intArg(args, "context_after", 0)
	maxResults := intArg(args, "max_results", 100)
	offset := intArg(args, "offset", 0)

	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}

	var results any
	if info.IsDir() {
		results, err = g.searchDirectory(ctx, root, pattern, globPattern, outputMode, contextBefore, contextAfter, maxResults, offset)
	} else {
		results, err = g.searchFile(root, pattern, outputMode, contextBefore, contextAfter, maxResults, offset)
	}

	if err != nil {
		return "", err
	}

	data, err := json.Marshal(results)
	if err != nil {
		return "", err
	}

	// Grep-specific 20KB limit with continuation support
	const maxGrepResultSize = 20000
	if len(data) > maxGrepResultSize {
		// Truncate and add continuation info
		truncated := data[:maxGrepResultSize]
		continuation := fmt.Sprintf("\n\n[TRUNCATED: Grep result too large (%d chars). Showing first %d chars. To see more results, use offset=%d to continue from where this left off.]", len(data), maxGrepResultSize, offset+maxResults)
		return string(truncated) + continuation, nil
	}

	return string(data), nil
}

func (g *GrepTool) searchDirectory(ctx context.Context, root string, pattern *regexp.Regexp, globPattern string, outputMode string, contextBefore, contextAfter, maxResults, offset int) (any, error) {
	type fileMatch struct {
		Path    string `json:"path"`
		Matches []any  `json:"matches,omitempty"`
		Count   int    `json:"count,omitempty"`
	}

	var results []fileMatch
	totalMatches := 0
	skippedMatches := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			return nil
		}

		if globPattern != "" {
			matched, err := filepath.Match(globPattern, filepath.Base(path))
			if err != nil || !matched {
				return nil
			}
		}

		if isBinaryFile(path) {
			return nil
		}

		relPath, _ := filepath.Rel(g.guard.root, path)

		matches, count := g.grepFile(path, pattern, outputMode, contextBefore, contextAfter, maxResults-totalMatches)
		if count > 0 {
			// Apply offset: skip matches until we've skipped enough
			if skippedMatches < offset {
				toSkip := offset - skippedMatches
				if toSkip >= count {
					// Skip entire file's matches
					skippedMatches += count
					return nil
				}
				// Partial skip: skip some matches from this file
				if outputMode == "content" && len(matches) > toSkip {
					matches = matches[toSkip:]
				}
				count -= toSkip
				skippedMatches += toSkip
			}

			fm := fileMatch{
				Path:  relPath,
				Count: count,
			}
			if outputMode == "content" {
				fm.Matches = matches
			}
			results = append(results, fm)
			totalMatches += count

			if totalMatches >= maxResults {
				return filepath.SkipAll
			}
		}

		return nil
	})

	if err != nil && !errors.Is(err, context.Canceled) {
		return nil, err
	}

	if outputMode == "files" {
		paths := make([]string, len(results))
		for i, r := range results {
			paths[i] = r.Path
		}
		return map[string]any{
			"count": len(paths),
			"files": paths,
		}, nil
	}

	return map[string]any{
		"count":   len(results),
		"results": results,
	}, nil
}

func (g *GrepTool) searchFile(path string, pattern *regexp.Regexp, outputMode string, contextBefore, contextAfter, maxResults, offset int) (any, error) {
	relPath, _ := filepath.Rel(g.guard.root, path)
	matches, count := g.grepFile(path, pattern, outputMode, contextBefore, contextAfter, maxResults+offset)

	// Apply offset to matches
	if offset > 0 && count > offset {
		if outputMode == "content" && len(matches) > offset {
			matches = matches[offset:]
		}
		count -= offset
	} else if offset > 0 {
		// Offset exceeds total matches
		count = 0
		matches = nil
	}

	if outputMode == "files" {
		if count > 0 {
			return map[string]any{
				"count": 1,
				"files": []string{relPath},
			}, nil
		}
		return map[string]any{
			"count": 0,
			"files": []string{},
		}, nil
	}

	return map[string]any{
		"path":    relPath,
		"count":   count,
		"matches": matches,
	}, nil
}

func (g *GrepTool) grepFile(path string, pattern *regexp.Regexp, outputMode string, contextBefore, contextAfter, maxResults int) ([]any, int) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0
	}
	defer file.Close()

	var matches []any
	count := 0

	if outputMode == "count" || outputMode == "files" {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if pattern.MatchString(scanner.Text()) {
				count++
			}
		}
		return nil, count
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	for i, line := range lines {
		lineNum = i + 1
		if pattern.MatchString(line) {
			var contextLines []grepLine

			start := i - contextBefore
			if start < 0 {
				start = 0
			}
			for j := start; j < i; j++ {
				contextLines = append(contextLines, grepLine{
					Line:    j + 1,
					Type:    "context",
					Content: lines[j],
				})
			}

			contextLines = append(contextLines, grepLine{
				Line:    lineNum,
				Type:    "match",
				Content: line,
			})

			end := i + contextAfter + 1
			if end > len(lines) {
				end = len(lines)
			}
			for j := i + 1; j < end; j++ {
				contextLines = append(contextLines, grepLine{
					Line:    j + 1,
					Type:    "context",
					Content: lines[j],
				})
			}

			matches = append(matches, contextLines)
			count++

			if count >= maxResults {
				break
			}
		}
	}

	return matches, count
}

func isBinaryFile(path string) bool {
	// Simple heuristic: check file extension
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".bin": true, ".dat": true, ".db": true, ".sqlite": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
	}
	return binaryExts[ext]
}
