package tooling

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ApplyPatchTool applies unified diff patches directly in Go.
type ApplyPatchTool struct {
	guard pathGuard
}

func NewApplyPatchTool(guard pathGuard) *ApplyPatchTool {
	return &ApplyPatchTool{guard: guard}
}

func (ApplyPatchTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "apply_patch",
			Description: "Apply one or more unified diff patches (*** Begin Patch blocks). Use read_file first to capture context, then submit the exact patch you want to apply.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"patch": map[string]any{
						"type":        "string",
						"description": "Unified diff patch text (*** Begin Patch / *** Update File sections).",
					},
				},
				"required": []string{"patch"},
			},
		},
	}
}

func (a *ApplyPatchTool) Call(ctx context.Context, args map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	patch, ok := stringArg(args, "patch")
	if !ok || strings.TrimSpace(patch) == "" {
		return "", errors.New("patch is required")
	}

	sections, err := a.parseSections(patch)
	if err != nil {
		return "", err
	}

	for _, section := range sections {
		switch section.op {
		case patchOpUpdate:
			if err := a.applyUpdate(section); err != nil {
				return "", err
			}
		case patchOpAdd:
			if err := a.applyAdd(section); err != nil {
				return "", err
			}
		case patchOpDelete:
			if err := a.applyDelete(section); err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("unknown patch op %q", section.op)
		}
	}

	return fmt.Sprintf("Applied %d patch block(s).", len(sections)), nil
}

const (
	patchOpUpdate = "update"
	patchOpAdd    = "add"
	patchOpDelete = "delete"
)

type patchSection struct {
	op   string
	path string
	body []string
}

func (a *ApplyPatchTool) parseSections(patch string) ([]patchSection, error) {
	scanner := bufio.NewScanner(strings.NewReader(patch))
	lineNum := 0
	var sections []patchSection
	var current *patchSection
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "*** Begin Patch"):
			if current != nil {
				return nil, fmt.Errorf("nested patch block at line %d", lineNum)
			}
			current = &patchSection{}
		case strings.HasPrefix(line, "*** End Patch"):
			if current == nil {
				return nil, fmt.Errorf("unexpected *** End Patch at line %d", lineNum)
			}
			if current.op == "" || current.path == "" {
				return nil, fmt.Errorf("patch block missing header before line %d", lineNum)
			}
			sections = append(sections, *current)
			current = nil
		case strings.HasPrefix(line, "*** Update File:"):
			if current == nil {
				return nil, fmt.Errorf("update header outside patch at line %d", lineNum)
			}
			current.op = patchOpUpdate
			current.path = strings.TrimSpace(strings.TrimPrefix(line, "*** Update File:"))
		case strings.HasPrefix(line, "*** Add File:"):
			if current == nil {
				return nil, fmt.Errorf("add header outside patch at line %d", lineNum)
			}
			current.op = patchOpAdd
			current.path = strings.TrimSpace(strings.TrimPrefix(line, "*** Add File:"))
		case strings.HasPrefix(line, "*** Delete File:"):
			if current == nil {
				return nil, fmt.Errorf("delete header outside patch at line %d", lineNum)
			}
			current.op = patchOpDelete
			current.path = strings.TrimSpace(strings.TrimPrefix(line, "*** Delete File:"))
		default:
			if current == nil {
				if strings.TrimSpace(line) == "" {
					continue
				}
				return nil, fmt.Errorf("unexpected line outside patch at %d", lineNum)
			}
			current.body = append(current.body, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if current != nil {
		return nil, errors.New("unterminated patch block")
	}

	if len(sections) == 0 {
		return nil, errors.New("no patch blocks found")
	}

	for i := range sections {
		if err := a.validatePath(sections[i].path); err != nil {
			return nil, err
		}
	}

	return sections, nil
}

func (a *ApplyPatchTool) validatePath(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return errors.New("patch path is empty")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("patch path %q must be relative", trimmed)
	}
	if strings.HasPrefix(filepath.Clean(trimmed), "..") {
		return fmt.Errorf("patch path %q escapes workspace", trimmed)
	}
	_, err := a.guard.Resolve(trimmed)
	return err
}

func (a *ApplyPatchTool) applyUpdate(section patchSection) error {
	absPath, err := a.guard.Resolve(section.path)
	if err != nil {
		return err
	}
	origData, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", section.path, err)
	}

	hunks, err := parseHunks(section.body)
	if err != nil {
		return fmt.Errorf("parse patch for %s: %w", section.path, err)
	}

	origLines, origHadTrailingNewline := splitLinesWithNewline(string(origData))
	newLines, err := applyHunks(origLines, hunks)
	if err != nil {
		return fmt.Errorf("apply patch for %s: %w", section.path, err)
	}

	content := joinLinesWithNewline(newLines, origHadTrailingNewline || hasTrailingNewlineFlag(section.body))
	return os.WriteFile(absPath, []byte(content), 0o644)
}

func (a *ApplyPatchTool) applyAdd(section patchSection) error {
	absPath, err := a.guard.Resolve(section.path)
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(absPath); statErr == nil {
		return fmt.Errorf("file %s already exists", section.path)
	}

	if containsHunkHeader(section.body) {
		hunks, err := parseHunks(section.body)
		if err != nil {
			return fmt.Errorf("parse patch for %s: %w", section.path, err)
		}
		newLines, err := applyHunks(nil, hunks)
		if err != nil {
			return fmt.Errorf("apply patch for %s: %w", section.path, err)
		}
		content := joinLinesWithNewline(newLines, hasTrailingNewlineFlag(section.body))
		return os.WriteFile(absPath, []byte(content), 0o644)
	}

	content := strings.Join(stripDiffPrefixes(section.body, '+'), "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(absPath, []byte(content), 0o644)
}

func (a *ApplyPatchTool) applyDelete(section patchSection) error {
	absPath, err := a.guard.Resolve(section.path)
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(absPath); statErr != nil {
		return fmt.Errorf("file %s does not exist", section.path)
	}
	return os.Remove(absPath)
}

// --- diff parsing helpers ---

type diffHunk struct {
	origStart int
	origCount int
	newStart  int
	newCount  int
	lines     []diffLine
}

type diffLine struct {
	op   byte
	text string
}

func parseHunks(lines []string) ([]diffHunk, error) {
	var hunks []diffHunk
	var current *diffHunk
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			if current != nil {
				hunks = append(hunks, *current)
			}
			h := diffHunk{}
			origStart, origCount, newStart, newCount, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			h.origStart, h.origCount = origStart, origCount
			h.newStart, h.newCount = newStart, newCount
			current = &h
			continue
		}
		if line == "\\ No newline at end of file" {
			continue
		}
		if current == nil {
			if strings.TrimSpace(line) == "" {
				continue
			}
			return nil, fmt.Errorf("unexpected content outside hunk: %s", line)
		}
		if line == "" {
			return nil, errors.New("empty hunk line")
		}
		op := line[0]
		if op != ' ' && op != '+' && op != '-' {
			return nil, fmt.Errorf("invalid hunk line: %s", line)
		}
		text := ""
		if len(line) > 1 {
			text = line[1:]
		}
		current.lines = append(current.lines, diffLine{op: op, text: text})
	}
	if current != nil {
		hunks = append(hunks, *current)
	}
	if len(hunks) == 0 {
		return nil, errors.New("no hunks found")
	}
	return hunks, nil
}

func parseHunkHeader(line string) (origStart, origCount, newStart, newCount int, err error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "@@") {
		return 0, 0, 0, 0, fmt.Errorf("invalid hunk header: %s", line)
	}
	line = strings.TrimPrefix(line, "@@")
	parts := strings.SplitN(line, "@@", 2)
	if len(parts) == 0 {
		return 0, 0, 0, 0, fmt.Errorf("invalid hunk header: %s", line)
	}
	ranges := strings.Fields(parts[0])
	if len(ranges) < 2 {
		return 0, 0, 0, 0, fmt.Errorf("invalid hunk header: %s", line)
	}
	origStart, origCount, err = parseRange(ranges[0])
	if err != nil {
		return
	}
	newStart, newCount, err = parseRange(ranges[1])
	return
}

func parseRange(rangeText string) (start, count int, err error) {
	rangeText = strings.TrimSpace(rangeText)
	if rangeText == "" {
		return 0, 0, errors.New("empty range")
	}
	prefix := rangeText[0]
	if prefix == '-' || prefix == '+' {
		rangeText = rangeText[1:]
	}
	parts := strings.Split(rangeText, ",")
	start, err = strconv.Atoi(parts[0])
	if err != nil {
		return
	}
	if len(parts) > 1 {
		count, err = strconv.Atoi(parts[1])
	} else {
		count = 1
	}
	return
}

func applyHunks(orig []string, hunks []diffHunk) ([]string, error) {
	result := make([]string, 0, len(orig))
	origIndex := 0
	for _, h := range hunks {
		start := h.origStart - 1
		if start < 0 {
			start = 0
		}
		if start > len(orig) {
			return nil, fmt.Errorf("hunk starts beyond EOF (line %d)", h.origStart)
		}
		if start < origIndex {
			return nil, fmt.Errorf("overlapping hunk at original line %d", h.origStart)
		}
		result = append(result, orig[origIndex:start]...)
		origIndex = start

		for _, l := range h.lines {
			switch l.op {
			case ' ':
				if origIndex >= len(orig) {
					return nil, errors.New("context line exceeds original length")
				}
				if orig[origIndex] != l.text {
					return nil, fmt.Errorf("context mismatch: expected %q, got %q", l.text, orig[origIndex])
				}
				result = append(result, orig[origIndex])
				origIndex++
			case '-':
				if origIndex >= len(orig) {
					return nil, errors.New("deletion exceeds original length")
				}
				if orig[origIndex] != l.text {
					return nil, fmt.Errorf("delete mismatch: expected %q, got %q", l.text, orig[origIndex])
				}
				origIndex++
			case '+':
				result = append(result, l.text)
			default:
				return nil, fmt.Errorf("unknown hunk op %q", l.op)
			}
		}
	}
	result = append(result, orig[origIndex:]...)
	return result, nil
}

func splitLinesWithNewline(input string) ([]string, bool) {
	if input == "" {
		return nil, false
	}
	hadNewline := strings.HasSuffix(input, "\n")
	trimmed := strings.TrimSuffix(input, "\n")
	if trimmed == "" {
		return []string{}, hadNewline
	}
	return strings.Split(trimmed, "\n"), hadNewline
}

func joinLinesWithNewline(lines []string, ensureNewline bool) string {
	if len(lines) == 0 {
		if ensureNewline {
			return "\n"
		}
		return ""
	}
	out := strings.Join(lines, "\n")
	if ensureNewline {
		out += "\n"
	}
	return out
}

func containsHunkHeader(lines []string) bool {
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			return true
		}
	}
	return false
}

func hasTrailingNewlineFlag(lines []string) bool {
	for _, line := range lines {
		if line == "\\ No newline at end of file" {
			return false
		}
	}
	return true
}

func stripDiffPrefixes(lines []string, expected byte) []string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(line) == 0 {
			result = append(result, line)
			continue
		}
		if line[0] == expected {
			result = append(result, line[1:])
		} else {
			result = append(result, line)
		}
	}
	return result
}
