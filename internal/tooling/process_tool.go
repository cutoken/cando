package tooling

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type BackgroundProcessTool struct {
	guard   pathGuard
	root    string
	binDir  string
	mu      sync.Mutex
	running map[string]*exec.Cmd
	rand    *rand.Rand
}

func NewBackgroundProcessTool(guard pathGuard, root string, binDir string) *BackgroundProcessTool {
	if root == "" {
		root = filepath.Join(guard.root, "processes")
	}
	if !filepath.IsAbs(root) {
		resolved, err := guard.Resolve(root)
		if err == nil {
			root = resolved
		}
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		panic(err)
	}
	return &BackgroundProcessTool{
		guard:   guard,
		root:    root,
		binDir:  binDir,
		running: make(map[string]*exec.Cmd),
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (t *BackgroundProcessTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "background_process",
			Description: "Manage long-running shell commands. Actions: start, list, logs, kill.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{
						"type":        "string",
						"description": "start | list | logs | kill",
					},
					"command": map[string]any{
						"type":        "array",
						"description": "Command + args (start action only).",
						"items":       map[string]any{"type": "string"},
					},
					"workdir": map[string]any{
						"type":        "string",
						"description": "Optional working directory relative to workspace root.",
					},
					"job_id": map[string]any{
						"type":        "string",
						"description": "Target job id for logs/kill.",
					},
					"stream": map[string]any{
						"type":        "string",
						"description": "stdout (default) or stderr for logs action.",
					},
					"tail_lines": map[string]any{
						"type":        "integer",
						"description": "Logs: number of lines from the end (default 50).",
					},
					"grep": map[string]any{
						"type":        "string",
						"description": "Logs: optional substring filter.",
					},
				},
				"required": []string{"action"},
			},
		},
	}
}

func (t *BackgroundProcessTool) Call(ctx context.Context, args map[string]any) (string, error) {
	action, ok := stringArg(args, "action")
	if !ok {
		return "", errors.New("action is required")
	}
	action = strings.ToLower(strings.TrimSpace(action))
	switch action {
	case "start":
		return t.handleStart(ctx, args)
	case "list":
		return t.handleList(args)
	case "logs":
		return t.handleLogs(args)
	case "kill":
		return t.handleKill(args)
	default:
		return "", fmt.Errorf("unknown action %s", action)
	}
}

func (t *BackgroundProcessTool) handleStart(ctx context.Context, args map[string]any) (string, error) {
	cmdArgs, err := stringSliceArg(args, "command")
	if err != nil {
		return "", err
	}
	if len(cmdArgs) == 0 {
		return "", errors.New("command must not be empty")
	}

	// Block commands that require user input
	blockedCommands := []string{"sudo", "su", "passwd"}
	cmdName := filepath.Base(cmdArgs[0])
	for _, blocked := range blockedCommands {
		if cmdName == blocked {
			return "", fmt.Errorf("command '%s' requires interactive input and is not allowed. Use alternative approaches that don't require user interaction", blocked)
		}
	}

	workdir := ""
	if wd, ok := stringArg(args, "workdir"); ok {
		workdir = wd
	}
	dir, err := t.guard.Resolve(workdir)
	if err != nil {
		return "", err
	}

	jobID := t.generateJobID()
	jobDir := filepath.Join(t.root, jobID)
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		return "", err
	}
	stdoutPath := filepath.Join(jobDir, "stdout.log")
	stderrPath := filepath.Join(jobDir, "stderr.log")
	metaPath := filepath.Join(jobDir, "meta.json")

	stdoutFile, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", fmt.Errorf("open stdout file: %w", err)
	}
	stderrFile, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		stdoutFile.Close()
		return "", fmt.Errorf("open stderr file: %w", err)
	}

	execCtx := context.Background()
	cmd := exec.CommandContext(execCtx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = dir
	cmd.Env = injectPath(os.Environ(), t.binDir)

	// Close stdin to prevent commands from hanging waiting for user input
	cmd.Stdin = nil

	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	meta := processMeta{
		ID:        jobID,
		Command:   cmdArgs,
		WorkDir:   dir,
		Status:    "running",
		StartedAt: time.Now(),
		Stdout:    stdoutPath,
		Stderr:    stderrPath,
	}
	if err := t.saveMeta(metaPath, &meta); err != nil {
		stdoutFile.Close()
		stderrFile.Close()
		return "", err
	}

	if err := cmd.Start(); err != nil {
		stdoutFile.Close()
		stderrFile.Close()
		meta.Status = "failed_start"
		meta.Error = err.Error()
		t.saveMeta(metaPath, &meta)
		return "", fmt.Errorf("start command: %w", err)
	}
	meta.PID = cmd.Process.Pid
	t.saveMeta(metaPath, &meta)

	t.mu.Lock()
	t.running[jobID] = cmd
	t.mu.Unlock()

	go func() {
		err := cmd.Wait()
		stdoutFile.Close()
		stderrFile.Close()
		meta.EndedAt = time.Now()
		if err != nil {
			meta.Status = "failed"
			meta.Error = err.Error()
			if exitErr, ok := err.(*exec.ExitError); ok {
				meta.ExitCode = exitErr.ExitCode()
			}
		} else {
			meta.Status = "exited"
			meta.ExitCode = 0
		}
		t.saveMeta(metaPath, &meta)
		t.mu.Lock()
		delete(t.running, jobID)
		t.mu.Unlock()
	}()

	resp, err := json.Marshal(map[string]any{
		"job_id":     jobID,
		"status":     meta.Status,
		"pid":        meta.PID,
		"started_at": meta.StartedAt.Format(time.RFC3339),
	})
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func (t *BackgroundProcessTool) handleList(args map[string]any) (string, error) {
	entries, err := os.ReadDir(t.root)
	if err != nil {
		if os.IsNotExist(err) {
			return "[]", nil
		}
		return "", err
	}
	type view struct {
		ID        string    `json:"job_id"`
		Command   []string  `json:"command"`
		WorkDir   string    `json:"workdir"`
		Status    string    `json:"status"`
		StartedAt time.Time `json:"started_at"`
		EndedAt   time.Time `json:"ended_at,omitempty"`
		ExitCode  int       `json:"exit_code"`
		PID       int       `json:"pid,omitempty"`
	}
	list := make([]view, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		meta, err := t.loadMeta(filepath.Join(t.root, entry.Name(), "meta.json"))
		if err != nil {
			continue
		}
		list = append(list, view{
			ID:        meta.ID,
			Command:   meta.Command,
			WorkDir:   meta.WorkDir,
			Status:    meta.Status,
			StartedAt: meta.StartedAt,
			EndedAt:   meta.EndedAt,
			ExitCode:  meta.ExitCode,
			PID:       meta.PID,
		})
	}
	// sort by start desc
	sort.Slice(list, func(i, j int) bool {
		return list[i].StartedAt.After(list[j].StartedAt)
	})
	resp, err := json.Marshal(list)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func (t *BackgroundProcessTool) handleLogs(args map[string]any) (string, error) {
	jobID, ok := stringArg(args, "job_id")
	if !ok || strings.TrimSpace(jobID) == "" {
		return "", errors.New("job_id is required")
	}
	meta, err := t.loadMeta(filepath.Join(t.root, jobID, "meta.json"))
	if err != nil {
		return "", err
	}
	stream := "stdout"
	if s, ok := stringArg(args, "stream"); ok && strings.TrimSpace(s) != "" {
		stream = strings.ToLower(strings.TrimSpace(s))
	}
	logPath := meta.Stdout
	if stream == "stderr" {
		logPath = meta.Stderr
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		return "", err
	}
	lines := splitLines(string(data))
	tail := intArg(args, "tail_lines", 50)
	if tail <= 0 || tail > len(lines) {
		tail = len(lines)
	}
	lines = lines[len(lines)-tail:]
	if pattern, ok := stringArg(args, "grep"); ok && strings.TrimSpace(pattern) != "" {
		filtered := lines[:0]
		for _, line := range lines {
			if strings.Contains(line, pattern) {
				filtered = append(filtered, line)
			}
		}
		lines = filtered
	}
	resp, err := json.Marshal(map[string]any{
		"job_id": jobID,
		"stream": stream,
		"lines":  lines,
	})
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func (t *BackgroundProcessTool) handleKill(args map[string]any) (string, error) {
	jobID, ok := stringArg(args, "job_id")
	if !ok || strings.TrimSpace(jobID) == "" {
		return "", errors.New("job_id is required")
	}
	t.mu.Lock()
	cmd := t.running[jobID]
	t.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return "", fmt.Errorf("job %s is not running", jobID)
	}
	if err := cmd.Process.Kill(); err != nil {
		return "", fmt.Errorf("kill failed: %w", err)
	}
	resp, err := json.Marshal(map[string]any{
		"job_id": jobID,
		"status": "killed",
	})
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func (t *BackgroundProcessTool) generateJobID() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return fmt.Sprintf("job-%d-%04x", time.Now().UnixNano(), t.rand.Intn(0xffff))
}

func (t *BackgroundProcessTool) saveMeta(path string, meta *processMeta) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (t *BackgroundProcessTool) loadMeta(path string) (*processMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta processMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

type processMeta struct {
	ID        string    `json:"id"`
	Command   []string  `json:"command"`
	WorkDir   string    `json:"workdir"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
	ExitCode  int       `json:"exit_code"`
	Error     string    `json:"error,omitempty"`
	PID       int       `json:"pid,omitempty"`
	Stdout    string    `json:"stdout"`
	Stderr    string    `json:"stderr"`
}

func splitLines(s string) []string {
	scanner := bufio.NewScanner(strings.NewReader(s))
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return lines
	}
	return lines
}
