package tooling

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cando/internal/credentials"
	"cando/internal/logging"
)

var errEntryLimit = errors.New("entry limit reached")

// Context helpers for passing session storage path to tools
type sessionStorageCtxKey struct{}

// WithSessionStorage adds the conversation storage path to context for session-aware tools
func WithSessionStorage(ctx context.Context, storagePath string) context.Context {
	return context.WithValue(ctx, sessionStorageCtxKey{}, storagePath)
}

// SessionStorageFromContext retrieves the conversation storage path from context
func SessionStorageFromContext(ctx context.Context) (string, bool) {
	path, ok := ctx.Value(sessionStorageCtxKey{}).(string)
	return path, ok
}

type ToolDefinition struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type Tool interface {
	Definition() ToolDefinition
	Call(ctx context.Context, args map[string]any) (string, error)
}

type Registry struct {
	tools       map[string]Tool
	definitions []ToolDefinition
}

func NewRegistry(tools ...Tool) *Registry {
	bucket := make(map[string]Tool, len(tools))
	defs := make([]ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		def := tool.Definition()
		bucket[def.Function.Name] = tool
		defs = append(defs, def)
	}
	return &Registry{tools: bucket, definitions: defs}
}

func (r *Registry) Definitions() []ToolDefinition {
	out := make([]ToolDefinition, len(r.definitions))
	copy(out, r.definitions)
	return out
}

func (r *Registry) Lookup(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *Registry) MustGet(name string) Tool {
	tool, ok := r.Lookup(name)
	if !ok {
		panic(fmt.Sprintf("tool %s is not registered", name))
	}
	return tool
}

type CredentialManager interface {
	Load() (*credentials.Credentials, error)
	Save(*credentials.Credentials) error
	Path() string
}

type Options struct {
	WorkspaceRoot string
	ShellTimeout  time.Duration
	PlanPath      string
	BinDir        string
	ExternalData  bool
	ProcessDir    string
	CredManager   CredentialManager
}

func DefaultTools(opts Options) []Tool {
	guard, err := newPathGuard(opts.WorkspaceRoot)
	if err != nil {
		panic(err)
	}
	planGuard := guard
	binDir := opts.BinDir
	switch {
	case binDir == "":
		binDir = filepath.Join(guard.root, "bin")
	case filepath.IsAbs(binDir):
		binDir = filepath.Clean(binDir)
	default:
		binDir, err = guard.Resolve(binDir)
		if err != nil {
			panic(err)
		}
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		panic(err)
	}
	planPath := opts.PlanPath
	if planPath == "" {
		planPath = filepath.Join(guard.root, "plan.json")
	} else {
		if !filepath.IsAbs(planPath) {
			resolved, err := guard.Resolve(planPath)
			if err != nil {
				panic(err)
			}
			planPath = resolved
		} else if opts.ExternalData {
			planPath = filepath.Clean(planPath)
			planGuard = pathGuard{root: filepath.Dir(planPath)}
		} else {
			resolved, err := guard.Resolve(planPath)
			if err != nil {
				panic(err)
			}
			planPath = resolved
		}
	}
	processDir := opts.ProcessDir
	if processDir == "" {
		processDir = filepath.Join(guard.root, "processes")
	}
	if err := os.MkdirAll(processDir, 0o755); err != nil {
		panic(err)
	}
	shellTimeout := opts.ShellTimeout
	if shellTimeout <= 0 {
		shellTimeout = 60 * time.Second
	}

	// Create background process tool first so it can be passed to shell tool
	bgTool := NewBackgroundProcessTool(guard, processDir, binDir)

	return []Tool{
		DateTimeTool{},
		WorkingDirectoryTool{root: guard.root},
		ListFilesTool{guard: guard},
		ReadFileTool{guard: guard},
		&ShellTool{
			guard:   guard,
			timeout: shellTimeout,
			binDir:  binDir,
			history: make(map[string]int),
			bgTool:  bgTool,
		},

		NewPlanToolWithGuard(planPath, planGuard),
		NewWebFetchJSONTool(shellTimeout),
		NewWriteFileTool(guard),
		NewEditFileTool(guard),
		NewApplyPatchTool(guard),
		NewGlobTool(guard),
		NewGrepTool(guard),
		NewVisionTool(guard, opts.CredManager),
		bgTool,
	}
}

type DateTimeTool struct{}

func (DateTimeTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "current_datetime",
			Description: "Return the user's current local date and time. Optional format override via Go time layout tokens.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"format": map[string]any{
						"type":        "string",
						"description": "Optional Go time layout (default RFC3339).",
					},
				},
			},
		},
	}
}

func (DateTimeTool) Call(ctx context.Context, args map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	layout := time.RFC3339
	if custom, ok := stringArg(args, "format"); ok && custom != "" {
		layout = custom
	}
	return time.Now().Format(layout), nil
}

type WorkingDirectoryTool struct {
	root string
}

func (w WorkingDirectoryTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "current_working_directory",
			Description: "Return the absolute workspace root configured for the agent.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
}

func (w WorkingDirectoryTool) Call(ctx context.Context, _ map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	return w.root, nil
}

type ListFilesTool struct {
	guard pathGuard
}

func (ListFilesTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "list_directory",
			Description: "List files within a directory, optionally recursively. All paths are constrained inside the workspace root.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Directory path to list (default workspace root).",
					},
					"recursive": map[string]any{
						"type":        "boolean",
						"description": "Whether to walk subdirectories.",
					},
					"include_hidden": map[string]any{
						"type":        "boolean",
						"description": "Include entries whose names start with '.'.",
					},
					"max_entries": map[string]any{
						"type":        "integer",
						"description": "Maximum number of entries to return (default 200).",
					},
				},
			},
		},
	}
}

func (l ListFilesTool) Call(ctx context.Context, args map[string]any) (string, error) {
	target := ""
	if provided, ok := stringArg(args, "path"); ok {
		target = provided
	}
	root, err := l.guard.Resolve(target)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", root)
	}
	includeHidden := boolArg(args, "include_hidden", false)
	recursive := boolArg(args, "recursive", false)
	maxEntries := intArg(args, "max_entries", 200)
	if maxEntries <= 0 {
		maxEntries = 200
	}

	type entry struct {
		Path string `json:"path"`
		Type string `json:"type"`
	}
	results := make([]entry, 0, maxEntries)
	truncated := false

	addEntry := func(path string, isDir bool) bool {
		if len(results) >= maxEntries {
			truncated = true
			return false
		}
		rel, relErr := filepath.Rel(l.guard.root, path)
		if relErr != nil {
			rel = path
		}
		if rel == "." {
			return true
		}
		name := filepath.Base(path)
		if !includeHidden && strings.HasPrefix(name, ".") {
			return true
		}
		results = append(results, entry{Path: rel, Type: typeOf(isDir)})
		return true
	}

	if recursive {
		walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if path == root {
				return nil
			}
			if !addEntry(path, d.IsDir()) {
				return errEntryLimit
			}
			return nil
		})
		if walkErr != nil && !errors.Is(walkErr, context.Canceled) && !errors.Is(walkErr, errEntryLimit) {
			return "", walkErr
		}
	} else {
		entries, err := os.ReadDir(root)
		if err != nil {
			return "", err
		}
		for _, e := range entries {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			default:
			}
			if !includeHidden && strings.HasPrefix(e.Name(), ".") {
				continue
			}
			if !addEntry(filepath.Join(root, e.Name()), e.IsDir()) {
				break
			}
		}
	}

	payload := map[string]any{
		"path":      root,
		"entries":   results,
		"truncated": truncated,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type ReadFileTool struct {
	guard pathGuard
}

func (ReadFileTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "read_file",
			Description: "Read a UTF-8 text file and return its contents (optionally truncated). The path must stay within the workspace root.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to the file to read, relative to the workspace root.",
					},
					"max_bytes": map[string]any{
						"type":        "integer",
						"description": "Maximum number of bytes to return (default 4096).",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (r ReadFileTool) Call(ctx context.Context, args map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	path, ok := stringArg(args, "path")
	if !ok || path == "" {
		return "", errors.New("path is required")
	}
	abs, err := r.guard.Resolve(path)
	if err != nil {
		return "", err
	}
	maxBytes := intArg(args, "max_bytes", 4096)
	if maxBytes <= 0 {
		maxBytes = 4096
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	truncated := false
	if len(data) > maxBytes {
		data = data[:maxBytes]
		truncated = true
	}
	rel, _ := filepath.Rel(r.guard.root, abs)
	payload := map[string]any{
		"path":      rel,
		"bytes":     len(data),
		"truncated": truncated,
		"content":   string(data),
	}
	out, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

type ShellTool struct {
	guard   pathGuard
	timeout time.Duration
	binDir  string
	history map[string]int
	hmu     sync.Mutex
	bgTool  *BackgroundProcessTool
}

func (s *ShellTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "shell",
			Description: "Execute commands within the workspace root. All file operations must stay inside the workspace tree. For long-running processes that don't exit (servers, watchers), use background=true.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"description": "Command to execute. Can be either an array of strings ['ls', '-la'] or a shell command string 'ls -la'.",
						"oneOf": []map[string]any{
							{
								"type": "array",
								"items": map[string]any{
									"type": "string",
								},
							},
							{
								"type": "string",
							},
						},
					},
					"workdir": map[string]any{
						"type":        "string",
						"description": "Working directory relative to the workspace root.",
					},
					"timeout_seconds": map[string]any{
						"type":        "number",
						"description": "Override the default timeout. Maximum 300 seconds (5 minutes).",
					},
					"background": map[string]any{
						"type":        "boolean",
						"description": "Run command in background. Returns job_id immediately. Use background_process tool to check logs/status or kill the job.",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

func (s *ShellTool) Call(ctx context.Context, args map[string]any) (string, error) {
	var rawCmd []string
	var err error

	// Try to get command as array first, then fall back to string parsing
	cmdRaw, ok := args["command"]
	if !ok {
		return "", errors.New("command is required")
	}

	switch v := cmdRaw.(type) {
	case []string:
		rawCmd = v
	case []any:
		rawCmd, err = stringSliceArg(args, "command")
		if err != nil {
			return "", err
		}
	case string:
		// Parse shell command string into arguments
		rawCmd, err = parseShellCommand(v)
		if err != nil {
			return "", fmt.Errorf("failed to parse command string: %w", err)
		}
	default:
		return "", errors.New("command must be an array of strings or a command string")
	}

	if len(rawCmd) == 0 {
		return "", errors.New("command must not be empty")
	}

	blockedCommands := []string{"sudo", "su", "passwd"}
	cmdName := filepath.Base(rawCmd[0])
	for _, blocked := range blockedCommands {
		if cmdName == blocked {
			logging.ErrorLog("shell: blocked command '%s' - interactive commands not allowed", blocked)
			return "", fmt.Errorf("command '%s' requires interactive input and is not allowed. Use alternative approaches that don't require user interaction", blocked)
		}
	}

	workdir := ""
	if provided, ok := stringArg(args, "workdir"); ok {
		workdir = provided
	}
	resolvedDir, err := s.guard.Resolve(workdir)
	if err != nil {
		return "", err
	}

	// Log command for debugging
	logging.DevLog("shell: executing command %v in %s", rawCmd, workdir)

	if bg, ok := args["background"].(bool); ok && bg {
		if s.bgTool == nil {
			return "", errors.New("background mode not available")
		}
		bgArgs := map[string]any{
			"action":  "start",
			"command": rawCmd,
		}
		if workdir != "" {
			bgArgs["workdir"] = workdir
		}
		return s.bgTool.Call(ctx, bgArgs)
	}

	key := s.commandKey(resolvedDir, rawCmd)
	count := s.recordCommand(key)
	var warning string
	switch {
	case count > 5:
		return "", errors.New("LLM went nuts repeating the same shell command")
	case count > 3:
		warning = fmt.Sprintf("What the fuck are you doing bro? This command has been repeated %d times.", count)
	}
	timeout := s.timeout
	if override, ok := args["timeout_seconds"]; ok {
		switch v := override.(type) {
		case float64:
			if v > 0 {
				timeout = time.Duration(v * float64(time.Second))
			}
		case int:
			if v > 0 {
				timeout = time.Duration(v) * time.Second
			}
		}
	}

	const maxShellTimeout = 300 * time.Second
	if timeout > maxShellTimeout {
		return "", fmt.Errorf("timeout_seconds cannot exceed 300 (5 minutes). For longer-running commands, use background=true")
	}

	ctxWithTimeout := ctx
	cancel := func() {}
	if timeout > 0 {
		ctxWithTimeout, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, rawCmd[0], rawCmd[1:]...)
	cmd.Dir = resolvedDir
	cmd.Env = injectPath(os.Environ(), s.binDir)

	cmd.Stdin = nil // prevent hangs on interactive input

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start)
	exitCode := 0
	if ps := cmd.ProcessState; ps != nil {
		exitCode = ps.ExitCode()
	}

	logging.DevLog("shell: command completed in %dms with exit code %d", duration.Milliseconds(), exitCode)

	result := map[string]any{
		"workdir":     resolvedDir,
		"stdout":      stdout.String(),
		"stderr":      stderr.String(),
		"exit_code":   exitCode,
		"duration_ms": duration.Milliseconds(),
	}
	if runErr != nil {
		if errors.Is(runErr, context.DeadlineExceeded) {
			logging.ErrorLog("shell: command timed out after %d seconds", int(timeout.Seconds()))
			result["error"] = fmt.Sprintf("Command timed out after %d seconds and was killed. Output may be incomplete.", int(timeout.Seconds()))
			result["timed_out"] = true
		} else {
			logging.ErrorLog("shell: command failed: %v", runErr)
			result["error"] = runErr.Error()
		}
	}
	if warning != "" {
		logging.ErrorLog("shell: %s", warning)
		result["warning"] = warning
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *ShellTool) commandKey(workdir string, cmd []string) string {
	return workdir + "|" + strings.Join(cmd, "\x00")
}

func (s *ShellTool) recordCommand(key string) int {
	s.hmu.Lock()
	defer s.hmu.Unlock()
	if s.history == nil {
		s.history = make(map[string]int)
	}
	s.history[key]++
	return s.history[key]
}

type PlanTool struct {
	path        string
	historyPath string
	mu          sync.Mutex
	guard       pathGuard
}

func NewPlanTool(path string) *PlanTool {
	return NewPlanToolWithGuard(path, pathGuard{root: filepath.Dir(path)})
}

func NewPlanToolWithGuard(path string, guard pathGuard) *PlanTool {
	if path == "" {
		path = "plan.json"
	}
	historyPath := path + ".history.json"
	return &PlanTool{path: path, historyPath: historyPath, guard: guard}
}

func (p *PlanTool) Definition() ToolDefinition {
	return ToolDefinition{
		Type: "function",
		Function: ToolFunction{
			Name:        "update_plan",
			Description: "Update or fetch the active execution plan persisted on disk. Use action=history to view the last updates.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{
						"type":        "string",
						"description": "Either 'update' (default), 'get', or 'history'.",
					},
					"steps": map[string]any{
						"type":        "array",
						"description": "List of plan steps when action is update.",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"status": map[string]any{
									"type":        "string",
									"description": "pending | in_progress | completed",
								},
								"step": map[string]any{
									"type":        "string",
									"description": "Description of the task step.",
								},
							},
							"required": []string{"status", "step"},
						},
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "For history action: maximum number of recent entries to return (default 10).",
					},
				},
			},
		},
	}
}

func (p *PlanTool) Call(ctx context.Context, args map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	planPath := p.path
	historyPath := p.historyPath
	if sessionStoragePath, ok := SessionStorageFromContext(ctx); ok && sessionStoragePath != "" {
		base := strings.TrimSuffix(sessionStoragePath, filepath.Ext(sessionStoragePath))
		planPath = base + "-plan.json"
		historyPath = base + "-plan.json.history.json"
	}

	action, _ := stringArg(args, "action")
	if action == "" {
		action = "update"
	}
	action = strings.ToLower(action)
	switch action {
	case "update":
		stepsRaw, ok := args["steps"]
		if !ok {
			return "", errors.New("steps are required for update action")
		}
		steps, err := parsePlanSteps(stepsRaw)
		if err != nil {
			return "", err
		}
		plan := planState{UpdatedAt: time.Now(), Steps: steps}
		if err := p.saveToPath(planPath, plan); err != nil {
			return "", err
		}
		if err := p.appendHistoryToPath(historyPath, plan); err != nil {
			return "", err
		}
		payload, err := json.Marshal(plan)
		if err != nil {
			return "", err
		}
		return string(payload), nil
	case "get":
		plan, err := p.loadFromPath(planPath)
		if err != nil {
			return "", err
		}
		payload, err := json.Marshal(plan)
		if err != nil {
			return "", err
		}
		return string(payload), nil
	case "history":
		limit := intArg(args, "limit", 10)
		if limit < 0 {
			limit = 0
		}
		entries, err := p.loadHistoryFromPath(historyPath, limit)
		if err != nil {
			return "", err
		}
		payload := map[string]any{
			"entries": entries,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("unknown action %s", action)
	}
}

func (p *PlanTool) save(plan planState) error {
	return p.saveToPath(p.path, plan)
}

func (p *PlanTool) saveToPath(path string, plan planState) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.ensureDir(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (p *PlanTool) load() (planState, error) {
	return p.loadFromPath(p.path)
}

func (p *PlanTool) loadFromPath(path string) (planState, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return planState{UpdatedAt: time.Time{}, Steps: []planStep{}}, nil
		}
		return planState{}, err
	}
	var plan planState
	if err := json.Unmarshal(data, &plan); err != nil {
		return planState{}, err
	}
	return plan, nil
}

func (p *PlanTool) appendHistory(entry planState) error {
	return p.appendHistoryToPath(p.historyPath, entry)
}

func (p *PlanTool) appendHistoryToPath(histPath string, entry planState) error {
	if histPath == "" {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.ensureDir(histPath); err != nil {
		return err
	}
	entries, err := p.readHistoryLockedFromPath(histPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		entries = []planState{}
	}
	entries = append(entries, entry)
	return p.writeHistoryLockedToPath(histPath, entries)
}

func (p *PlanTool) loadHistory(limit int) ([]planState, error) {
	return p.loadHistoryFromPath(p.historyPath, limit)
}

func (p *PlanTool) loadHistoryFromPath(histPath string, limit int) ([]planState, error) {
	if histPath == "" {
		return []planState{}, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	entries, err := p.readHistoryLockedFromPath(histPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []planState{}, nil
		}
		return nil, err
	}
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries, nil
}

func (p *PlanTool) readHistoryLocked() ([]planState, error) {
	return p.readHistoryLockedFromPath(p.historyPath)
}

func (p *PlanTool) readHistoryLockedFromPath(histPath string) ([]planState, error) {
	data, err := os.ReadFile(histPath)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []planState{}, nil
	}
	var entries []planState
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (p *PlanTool) writeHistoryLocked(entries []planState) error {
	return p.writeHistoryLockedToPath(p.historyPath, entries)
}

func (p *PlanTool) writeHistoryLockedToPath(histPath string, entries []planState) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(histPath, data, 0o644)
}

func (p *PlanTool) ensureDir(path string) error {
	parent := filepath.Dir(path)
	if parent == "" || parent == "." {
		parent = p.guard.root
	}
	if parent == "" {
		return nil
	}
	return os.MkdirAll(parent, 0o755)
}

type planState struct {
	UpdatedAt time.Time  `json:"updated_at"`
	Steps     []planStep `json:"steps"`
}

type planStep struct {
	Status string `json:"status"`
	Step   string `json:"step"`
}

func parsePlanSteps(raw any) ([]planStep, error) {
	list, ok := raw.([]any)
	if !ok {
		if typed, ok := raw.([]map[string]any); ok {
			list = make([]any, len(typed))
			for i := range typed {
				list[i] = typed[i]
			}
		} else {
			return nil, errors.New("steps must be an array")
		}
	}
	steps := make([]planStep, 0, len(list))
	for idx, item := range list {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("step %d is not an object", idx)
		}
		status, ok := stringArg(obj, "status")
		if !ok {
			return nil, fmt.Errorf("step %d missing status", idx)
		}
		status = strings.ToLower(strings.TrimSpace(status))
		if status != "pending" && status != "in_progress" && status != "completed" {
			return nil, fmt.Errorf("step %d has invalid status %s", idx, status)
		}
		desc, ok := stringArg(obj, "step")
		if !ok || strings.TrimSpace(desc) == "" {
			return nil, fmt.Errorf("step %d missing description", idx)
		}
		steps = append(steps, planStep{Status: status, Step: desc})
	}
	return steps, nil
}

type pathGuard struct {
	root string
}

func newPathGuard(root string) (pathGuard, error) {
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return pathGuard{}, err
	}
	return pathGuard{root: abs}, nil
}

func (p pathGuard) Resolve(path string) (string, error) {
	var target string
	if path == "" {
		target = p.root
	} else if filepath.IsAbs(path) {
		target = path
	} else {
		target = filepath.Join(p.root, path)
	}
	cleaned, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if cleaned != p.root && !strings.HasPrefix(cleaned, p.root+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %s escapes workspace root", path)
	}
	return cleaned, nil
}

func (p pathGuard) Rel(path string) string {
	rel, err := filepath.Rel(p.root, path)
	if err != nil {
		return path
	}
	return rel
}

func injectPath(env []string, binDir string) []string {
	if binDir == "" {
		return env
	}
	pathPrefix := fmt.Sprintf("PATH=%s", binDir)
	for i, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			orig := strings.TrimPrefix(kv, "PATH=")
			env[i] = fmt.Sprintf("PATH=%s:%s", binDir, orig)
			return env
		}
	}
	return append(env, pathPrefix)
}

func stringSliceArg(args map[string]any, key string) ([]string, error) {
	raw, ok := args[key]
	if !ok {
		return nil, fmt.Errorf("%s is required", key)
	}
	switch v := raw.(type) {
	case []string:
		if len(v) == 0 {
			return nil, fmt.Errorf("%s is empty", key)
		}
		return v, nil
	case []any:
		out := make([]string, 0, len(v))
		for idx, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s[%d] is not a string", key, idx)
			}
			out = append(out, str)
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("%s is empty", key)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%s must be an array of strings", key)
	}
}

func stringArg(args map[string]any, key string) (string, bool) {
	val, ok := args[key]
	if !ok {
		return "", false
	}
	switch cast := val.(type) {
	case string:
		return cast, true
	default:
		return fmt.Sprintf("%v", cast), true
	}
}

func boolArg(args map[string]any, key string, defaultVal bool) bool {
	val, ok := args[key]
	if !ok {
		return defaultVal
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return defaultVal
}

func intArg(args map[string]any, key string, defaultVal int) int {
	val, ok := args[key]
	if !ok {
		return defaultVal
	}
	switch n := val.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return defaultVal
	}
}

func typeOf(isDir bool) string {
	if isDir {
		return "directory"
	}
	return "file"
}

func parseShellCommand(cmd string) ([]string, error) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil, errors.New("command string is empty")
	}

	var args []string
	var current strings.Builder
	var inQuote rune
	escaped := false

	for _, ch := range cmd {
		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		if inQuote != 0 {
			if ch == inQuote {
				inQuote = 0
			} else {
				current.WriteRune(ch)
			}
			continue
		}

		if ch == '"' || ch == '\'' {
			inQuote = ch
			continue
		}

		if ch == ' ' || ch == '\t' {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(ch)
	}

	if inQuote != 0 {
		return nil, fmt.Errorf("unclosed quote in command")
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	if len(args) == 0 {
		return nil, errors.New("no arguments parsed from command string")
	}

	return args, nil
}
