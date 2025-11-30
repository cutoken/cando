package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	prompt "github.com/c-bata/go-prompt"
	"github.com/charmbracelet/glamour"
	"golang.org/x/term"

	"cando/internal/config"
	"cando/internal/contextprofile"
	"cando/internal/credentials"
	"cando/internal/llm"
	"cando/internal/logging"
	"cando/internal/prompts"
	"cando/internal/state"
	"cando/internal/tooling"
)

var commandSuggestions = []prompt.Suggest{
	{Text: ":help", Description: "show this text"},
	{Text: ":states", Description: "list known conversation keys"},
	{Text: ":use", Description: "switch to an existing state"},
	{Text: ":new", Description: "create and switch to a blank state"},
	{Text: ":clear", Description: "wipe the current state's history"},
	{Text: ":drop", Description: "delete a stored state"},
	{Text: ":tools", Description: "list registered tools"},
	{Text: ":memories", Description: "inspect stored memories"},
	{Text: ":compact", Description: "force a compaction pass (:compact [protect_count])"},
	{Text: ":thinking", Description: "toggle thinking mode (:thinking on|off)"},
	{Text: ":reload", Description: "reload config (optionally provide path)"},
	{Text: ":quit", Description: "exit the program"},
	{Text: ":exit", Description: "exit the program"},
}

type interruptTracker struct {
	mu     sync.Mutex
	last   time.Time
	window time.Duration
}

func newInterruptTracker(window time.Duration) *interruptTracker {
	return &interruptTracker{window: window}
}

func (t *interruptTracker) secondPress() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	if !t.last.IsZero() && now.Sub(t.last) < t.window {
		t.last = time.Time{}
		return true
	}
	t.last = now
	return false
}

type promptExit struct{}

// WorkspaceContext holds state, tools, and profile for a specific workspace
type WorkspaceContext struct {
	states   *state.Manager
	tools    *tooling.Registry
	profile  contextprofile.Profile
	root     string
	planMode bool // When true, LLM is instructed to only plan/analyze, not make changes
}

// loadProjectInstructions reads the project instructions file for a workspace.
// Returns empty string if no instructions file exists.
func loadProjectInstructions(workspaceRoot string) string {
	storageRoot, err := ProjectStorageRoot(workspaceRoot)
	if err != nil {
		return ""
	}
	instructionsPath := filepath.Join(storageRoot, "instructions.txt")
	content, err := os.ReadFile(instructionsPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

// injectProjectInstructions modifies messages to append project instructions to the system message
func injectProjectInstructions(messages []state.Message, instructions string) []state.Message {
	if instructions == "" || len(messages) == 0 {
		return messages
	}

	// Make a copy to avoid modifying the original
	result := make([]state.Message, len(messages))
	copy(result, messages)

	// Find and modify the system message (usually the first one)
	for i, msg := range result {
		if msg.Role == "system" {
			result[i].Content = msg.Content + "\n\n---\nProject Instructions:\n" + instructions
			break
		}
	}
	return result
}

// loadProjectFacts reads project facts from the workspace storage.
// Returns empty slice if no facts file exists.
func loadProjectFacts(workspaceRoot string) []string {
	if workspaceRoot == "" {
		return nil
	}
	storageRoot, err := ProjectStorageRoot(workspaceRoot)
	if err != nil {
		return nil
	}
	factsPath := filepath.Join(storageRoot, "project_facts.json")
	content, err := os.ReadFile(factsPath)
	if err != nil {
		return nil
	}
	var facts []string
	if err := json.Unmarshal(content, &facts); err != nil {
		return nil
	}
	return facts
}

// saveProjectFacts writes project facts to the workspace storage.
func saveProjectFacts(workspaceRoot string, facts []string) error {
	if workspaceRoot == "" {
		return fmt.Errorf("no workspace root")
	}
	storageRoot, err := ProjectStorageRoot(workspaceRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(storageRoot, 0o755); err != nil {
		return err
	}
	factsPath := filepath.Join(storageRoot, "project_facts.json")
	content, err := json.MarshalIndent(facts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(factsPath, content, 0o644)
}

// planModeHint is the instruction appended when plan mode is enabled
const planModeHint = `

---
PLAN MODE ENABLED: The user has enabled plan/analysis mode. You should ONLY:
- Analyze code and explain what you find
- Plan and outline changes without implementing them
- Answer questions and provide recommendations
- Research and investigate

DO NOT make any file changes. If the user asks you to implement something, politely remind them that plan mode is enabled and they should turn it off if they want you to make changes.`

// injectPlanModeHint appends the plan mode instruction to the system message
func injectPlanModeHint(messages []state.Message) []state.Message {
	if len(messages) == 0 {
		return messages
	}

	// Make a copy to avoid modifying the original
	result := make([]state.Message, len(messages))
	copy(result, messages)

	// Find and modify the system message
	for i, msg := range result {
		if msg.Role == "system" {
			result[i].Content = msg.Content + planModeHint
			break
		}
	}
	return result
}

// injectProjectFacts modifies messages to append project facts to the system message
func injectProjectFacts(messages []state.Message, facts []string) []state.Message {
	if len(facts) == 0 || len(messages) == 0 {
		return messages
	}

	// Make a copy to avoid modifying the original
	result := make([]state.Message, len(messages))
	copy(result, messages)

	// Format facts as bullet points
	var factsText strings.Builder
	for _, fact := range facts {
		factsText.WriteString("- ")
		factsText.WriteString(fact)
		factsText.WriteString("\n")
	}

	// Find and modify the system message (usually the first one)
	for i, msg := range result {
		if msg.Role == "system" {
			result[i].Content = msg.Content + "\n\n---\nProject Facts (learned from previous sessions):\n" + factsText.String()
			break
		}
	}
	return result
}

// projectFactsExtractor implements contextprofile.FactsExtractor
type projectFactsExtractor struct {
	client        llm.Client
	model         string
	workspaceRoot string
	logger        *log.Logger
}

// ExtractFacts extracts project facts from the conversation before compaction
func (e *projectFactsExtractor) ExtractFacts(ctx context.Context, messages []state.Message) error {
	if e.workspaceRoot == "" || e.client == nil {
		return nil
	}

	// Load existing facts
	existingFacts := loadProjectFacts(e.workspaceRoot)

	// Build the conversation content for extraction
	var convBuilder strings.Builder
	for _, msg := range messages {
		if msg.Role == "system" {
			continue // Skip system messages
		}
		convBuilder.WriteString(fmt.Sprintf("[%s]: %s\n\n", msg.Role, msg.Content))
	}

	// Build user message with conversation and existing facts
	existingFactsJSON, _ := json.Marshal(existingFacts)
	userContent := fmt.Sprintf("Conversation:\n%s\n\nExisting facts:\n%s", convBuilder.String(), string(existingFactsJSON))

	// Make LLM call to extract facts
	resp, err := e.client.Chat(ctx, llm.ChatRequest{
		Model: e.model,
		Messages: []state.Message{
			{Role: "system", Content: prompts.FactsExtraction()},
			{Role: "user", Content: userContent},
		},
		Temperature: 0.3,
	})
	if err != nil {
		return fmt.Errorf("facts extraction LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("no response from LLM")
	}

	// Parse JSON response
	responseText := strings.TrimSpace(resp.Choices[0].Message.Content)
	var newFacts []string
	if err := json.Unmarshal([]byte(responseText), &newFacts); err != nil {
		// Try to extract JSON from response if it has extra text
		start := strings.Index(responseText, "[")
		end := strings.LastIndex(responseText, "]")
		if start >= 0 && end > start {
			if err := json.Unmarshal([]byte(responseText[start:end+1]), &newFacts); err != nil {
				e.logger.Printf("failed to parse facts response: %v", err)
				return nil // Don't fail on parse errors
			}
		} else {
			e.logger.Printf("failed to parse facts response: %v", err)
			return nil
		}
	}

	// Limit to ~200 facts max
	if len(newFacts) > 200 {
		newFacts = newFacts[:200]
	}

	// Save updated facts
	if err := saveProjectFacts(e.workspaceRoot, newFacts); err != nil {
		return fmt.Errorf("failed to save facts: %w", err)
	}

	e.logger.Printf("extracted %d project facts", len(newFacts))
	return nil
}

// Agent wires the CLI, state machine, tools, and LLM client together.
type Agent struct {
	client           llm.Client
	cfg              config.Config
	cfgPath          string
	providerCtrl     ProviderSwitcher
	states           *state.Manager // Default workspace state (for CLI mode)
	systemPrompt     string
	profile          contextprofile.Profile
	tools            *tooling.Registry // Default workspace tools (for CLI mode)
	logger           *log.Logger
	credManager      CredentialManager
	providerBuilders map[string]ProviderBuilder
	isTTY            bool
	render           *glamour.TermRenderer
	requestCancelMu  sync.Mutex
	requestCancel    context.CancelFunc
	planMu           sync.RWMutex
	lastPlan         *planSnapshot
	sessionOnce      sync.Once
	sessionOnceErr   error
	resumeKey        string
	tokenMu          sync.RWMutex
	workspaceRoot    string // Default workspace (for CLI mode)
	totalTokens      int
	toolOpts         tooling.Options // Original tool options for workspace switching
	activeProvider   string          // Provider name for creating workspace profiles
	profileModel     string          // Model name for creating workspace profiles
	version          string          // Application version for update checks

	// Multi-workspace support for web mode
	workspacesMu      sync.RWMutex
	workspaceContexts map[string]*WorkspaceContext // workspace path -> context
}

// CredentialManager interface for credential operations
type CredentialManager interface {
	Load() (*credentials.Credentials, error)
	Save(*credentials.Credentials) error
	Path() string
}

type ProviderBuilder func(cfg config.Config, apiKey string, logger *log.Logger) (*ProviderRegistration, error)

type Options struct {
	ResumeKey        string
	WorkspaceRoot    string
	ProviderBuilders map[string]ProviderBuilder
	ActiveProvider   string // Provider name for creating workspace profiles
	ProfileModel     string // Model name for creating workspace profiles
	Version          string // Application version for update checks
}

// New returns a fully wired Agent ready for the REPL loop.
func New(client llm.Client, cfg config.Config, cfgPath string, mgr *state.Manager, profile contextprofile.Profile, registry *tooling.Registry, logger *log.Logger, credMgr CredentialManager, opts Options, toolOpts tooling.Options) *Agent {
	var renderer *glamour.TermRenderer
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(0),
		); err == nil {
			renderer = r
		}
	}

	agent := &Agent{
		client:            client,
		cfg:               cfg,
		cfgPath:           cfgPath,
		providerCtrl:      providerCtrlForClient(client),
		states:            mgr,
		systemPrompt:      prompts.Combine(strings.TrimSpace(cfg.SystemPrompt)),
		profile:           profile,
		tools:             registry,
		logger:            logger,
		credManager:       credMgr,
		providerBuilders:  opts.ProviderBuilders,
		isTTY:             term.IsTerminal(int(os.Stdin.Fd())),
		render:            renderer,
		resumeKey:         strings.TrimSpace(opts.ResumeKey),
		workspaceRoot:     opts.WorkspaceRoot,
		toolOpts:          toolOpts,
		activeProvider:    opts.ActiveProvider,
		profileModel:      opts.ProfileModel,
		version:           opts.Version,
		workspaceContexts: make(map[string]*WorkspaceContext),
	}

	if agent.providerCtrl != nil {
		if opt := agent.providerCtrl.ActiveProvider(); opt.Model != "" {
			agent.cfg.Model = opt.Model
		}
	}

	if agent.states != nil {
		agent.states.SetSystemPrompt(agent.systemPrompt)
	}

	return agent
}

func providerCtrlForClient(client llm.Client) ProviderSwitcher {
	if client == nil {
		return nil
	}
	if ctrl, ok := client.(ProviderSwitcher); ok {
		return ctrl
	}
	return nil
}

// Run starts the CLI prompt and blocks until the session finishes.
func (a *Agent) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tracker := newInterruptTracker(2 * time.Second)
	if a.isTTY {
		return a.runPrompt(ctx, cancel, tracker)
	}
	go a.handleInterrupts(ctx, cancel, tracker)
	return a.runNonInteractive(ctx, cancel)
}

// RunOneShot executes a single prompt and returns the response
func (a *Agent) RunOneShot(ctx context.Context, prompt string) error {
	if err := a.ensureSessionSelected(); err != nil {
		return fmt.Errorf("ensure session: %w", err)
	}

	response, finishReason, err := a.respond(ctx, prompt)
	if err != nil {
		return fmt.Errorf("respond: %w", err)
	}

	if response != "" {
		a.printResponse(response)
	}

	if finishReason == "stop" {
		fmt.Println("(Model emitted stop)")
	}

	return nil
}

func (a *Agent) reloadConfig(path string) error {
	newCfg, err := config.Load(path)
	if err != nil {
		logging.ErrorLog("failed to reload config from %s: %v", path, err)
		return err
	}
	if !strings.EqualFold(newCfg.Provider, a.cfg.Provider) {
		return fmt.Errorf("provider changes require restart (current %s vs %s)", a.cfg.Provider, newCfg.Provider)
	}
	if !strings.EqualFold(newCfg.ContextProfile, a.cfg.ContextProfile) {
		fmt.Println("Warning: context_profile changes require restart to take effect.")
	}
	if newCfg.WorkspaceRoot != a.cfg.WorkspaceRoot {
		fmt.Println("Warning: workspace_root changes require restart.")
	}
	if reloader, ok := a.profile.(contextprofile.ConfigReloadable); ok {
		if err := reloader.ReloadConfig(newCfg); err != nil {
			logging.ErrorLog("context profile reload failed: %v", err)
			return err
		}
	}
	a.cfg = newCfg
	a.cfgPath = path
	logging.UserLog("Config reloaded from %s", path)
	return nil
}

func (a *Agent) runPrompt(ctx context.Context, cancel context.CancelFunc, tracker *interruptTracker) (err error) {
	fmt.Println("👋 Welcome to Cando! Your AI assistant is ready to help.")
	fmt.Println("Type ':help' for commands. Send prompts to talk to the agent. Use double Ctrl+C to exit.")

	if err := a.ensureSessionSelected(); err != nil {
		return err
	}

	current := a.states.Current()
	if msgs := current.Messages(); len(msgs) > 0 {
		fmt.Printf("(loaded %d conversation messages totaling %d chars)\n", len(msgs), conversationCharCount(msgs))
	}

	history := loadInputHistory(a.cfg.HistoryPath)

	var restore func()
	if fd := int(os.Stdin.Fd()); term.IsTerminal(fd) {
		if state, terr := term.GetState(fd); terr == nil {
			restore = func() { _ = term.Restore(fd, state) }
		}
	}
	if restore != nil {
		defer restore()
	}

	var exitRequested atomic.Bool
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(promptExit); ok {
				err = nil
				return
			}
			panic(r)
		}
	}()

	executor := func(in string) {
		if exitRequested.Load() || ctx.Err() != nil {
			return
		}
		line := strings.TrimSpace(in)
		if line == "" {
			return
		}
		history.Add(line)
		if exit := a.handleLine(ctx, line); exit {
			exitRequested.Store(true)
			cancel()
			panic(promptExit{})
		}
	}

	p := prompt.New(
		executor,
		a.commandCompleter(),
		prompt.OptionHistory(history.Entries()),
		prompt.OptionTitle("Cando"),
		prompt.OptionLivePrefix(func() (string, bool) {
			current := a.states.Current()
			return fmt.Sprintf("[%s] > ", current.Key()), true
		}),
		prompt.OptionAddKeyBind(
			prompt.KeyBind{
				Key: prompt.ControlC,
				Fn: func(buf *prompt.Buffer) {
					if a.cancelInFlightRequest() {
						fmt.Println("\n(Current request cancelled.)")
						return
					}
					second := tracker.secondPress()
					if second {
						fmt.Println("\nReceived second Ctrl+C, exiting.")
						exitRequested.Store(true)
						cancel()
						panic(promptExit{})
					}
					fmt.Println("\n(Press Ctrl+C again within 2s to exit)")
				},
			},
			prompt.KeyBind{
				Key: prompt.ControlD,
				Fn: func(buf *prompt.Buffer) {
					if buf.Text() == "" {
						exitRequested.Store(true)
						cancel()
						panic(promptExit{})
					}
				},
			},
			prompt.KeyBind{
				Key: prompt.Escape,
				Fn: func(buf *prompt.Buffer) {
					if a.cancelInFlightRequest() {
						fmt.Println("\n(Request cancelled.)")
					}
				},
			},
		),
		prompt.OptionSetExitCheckerOnInput(func(string, bool) bool {
			if exitRequested.Load() {
				return true
			}
			select {
			case <-ctx.Done():
				return true
			default:
				return false
			}
		}),
	)

	p.Run()
	return nil
}

func (a *Agent) commandCompleter() func(prompt.Document) []prompt.Suggest {
	return func(doc prompt.Document) []prompt.Suggest {
		word := doc.GetWordBeforeCursor()
		prefix := strings.TrimLeft(doc.TextBeforeCursor(), " \t")
		if !strings.HasPrefix(prefix, ":") {
			return nil
		}
		return prompt.FilterHasPrefix(commandSuggestions, word, true)
	}
}

func (a *Agent) runNonInteractive(ctx context.Context, cancel context.CancelFunc) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("👋 Welcome to Cando! Your AI assistant is ready to help.")
	fmt.Println("Type ':help' for commands. Send prompts to talk to the agent. Use double Ctrl+C to exit.")

	if err := a.ensureSessionSelected(); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		current := a.states.Current()
		fmt.Printf("[%s] > ", current.Key())

		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println()
				return nil
			}
			if ctx.Err() != nil {
				fmt.Println()
				return nil
			}
			return fmt.Errorf("read input: %w", err)
		}
		if exit := a.handleLine(ctx, trimLineEnding(line)); exit {
			cancel()
			return nil
		}
	}
}

func (a *Agent) handleInterrupts(ctx context.Context, cancel context.CancelFunc, tracker *interruptTracker) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			if tracker.secondPress() {
				fmt.Println("\nReceived second Ctrl+C, exiting.")
				cancel()
				return
			}
			fmt.Println("\n(Press Ctrl+C again within 2s to exit)")
		}
	}
}

func (a *Agent) handleLine(ctx context.Context, input string) bool {
	trimmedLeft := strings.TrimLeft(input, " \t")
	if trimmedLeft == "" {
		return false
	}

	if strings.HasPrefix(trimmedLeft, ":") {
		return a.handleCommand(strings.TrimSpace(input))
	}

	// Log user input for debugging
	logging.DevLog("dispatching prompt: %d chars", len(input))
	response, finishReason, err := a.respond(ctx, input)
	logging.DevLog("response received: err=%v finish=%s len=%d", err, finishReason, len(response))
	if err != nil {
		logging.ErrorLog("agent error: %v", err)
		return false
	}
	if response != "" {
		a.printResponse(response)
	}
	if finishReason == "stop" {
		fmt.Println("(Model emitted stop; awaiting next prompt.)")
	}
	return false
}

func (a *Agent) respond(ctx context.Context, userInput string) (string, string, error) {
	conv := a.states.Current()
	conv.Append(state.Message{Role: "user", Content: userInput})
	if err := a.states.Save(conv); err != nil {
		return "", "", fmt.Errorf("save conversation: %w", err)
	}
	return a.respondLoopCLI(ctx, conv, a.states)
}

func (a *Agent) respondLoopCLI(ctx context.Context, conv *state.Conversation, stateManager *state.Manager) (string, string, error) {
	for {
		prepared, err := a.profile.Prepare(ctx, conv)
		if err != nil {
			logging.DevLog("context profile prepare failed: %v", err)
		}
		if prepared.Mutated {
			if err := stateManager.Save(conv); err != nil {
				return "", "", fmt.Errorf("save conversation: %w", err)
			}
		}
		messages := prepared.Messages
		if len(messages) == 0 {
			messages = conv.Messages()
		}

		// Inject hidden ultrathink message when force thinking is enabled
		// Only inject for user messages, not for tool call response rounds
		requestMessages := messages
		if a.cfg.ForceThinking && len(messages) > 0 && messages[len(messages)-1].Role == "user" {
			requestMessages = make([]state.Message, len(messages), len(messages)+1)
			copy(requestMessages, messages)
			requestMessages = append(requestMessages, state.Message{
				Role:    "user",
				Content: "ultrathink think very hard. reason step by step before answering.",
			})
		}

		totalChars := conversationCharCount(messages)
		logging.DevLog("invoking provider with %d messages (~%d chars)", len(messages), totalChars)
		fmt.Printf("(context size: %d chars)\n", totalChars)
		req := llm.ChatRequest{
			Model:       a.getActiveModel(),
			Messages:    requestMessages,
			Tools:       a.tools.Definitions(),
			Temperature: a.cfg.Temperature,
			Thinking: func() *llm.ThinkingOptions {
				if !a.cfg.ThinkingEnabled {
					return nil
				}
				return &llm.ThinkingOptions{Type: "enabled"}
			}(),
		}

		reqCtx, reqCancel := context.WithCancel(ctx)
		a.setInFlightCancel(reqCancel)
		resp, err := a.callProviderWithRetry(reqCtx, req, nil)
		a.clearInFlightCancel()
		reqCancel()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Println("(request cancelled)")
				return "", "", nil
			}
			return "", "", fmt.Errorf("chat completion: %w", err)
		}
		logging.DevLog("received %d choices", len(resp.Choices))
		if resp.Usage != nil {
			logging.DevLog("token usage: prompt=%d completion=%d total=%d",
				resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
			a.addTokens(resp.Usage.TotalTokens)
		}
		if len(resp.Choices) == 0 {
			return "", "", fmt.Errorf("no choices returned")
		}

		choice := resp.Choices[0]
		if len(choice.Message.ToolCalls) > 0 {
			// Tool calls will be processed separately
		}
		conv.Append(choice.Message)
		if err := stateManager.Save(conv); err != nil {
			return "", "", fmt.Errorf("save conversation: %w", err)
		}

		if len(choice.Message.ToolCalls) == 0 {
			if mutated, err := a.profile.AfterResponse(ctx, conv); err != nil {
				logging.DevLog("context profile after-response failed: %v", err)
			} else if mutated {
				if err := stateManager.Save(conv); err != nil {
					return "", "", fmt.Errorf("save conversation: %w", err)
				}
			}
			return choice.Message.Content, choice.FinishReason, nil
		}

		if err := a.processToolCallsWithCallback(ctx, conv, choice.Message.ToolCalls, nil, stateManager, a.tools, false); err != nil {
			return "", "", err
		}
		if mutated, err := a.profile.AfterResponse(ctx, conv); err != nil {
			logging.DevLog("context profile after-response failed: %v", err)
		} else if mutated {
			if err := stateManager.Save(conv); err != nil {
				return "", "", fmt.Errorf("save conversation: %w", err)
			}
		}
	}
}

type StreamCallback func(eventType string, data any) error

// respondWithCallbacksForWorkspace executes a conversation turn using a specific workspace context
func (a *Agent) respondWithCallbacksForWorkspace(ctx context.Context, userInput string, callback StreamCallback, wsCtx *WorkspaceContext) (string, string, error) {
	conv := wsCtx.states.Current()
	conv.Append(state.Message{Role: "user", Content: userInput})
	if err := wsCtx.states.Save(conv); err != nil {
		return "", "", fmt.Errorf("save conversation: %w", err)
	}

	// Wire up compaction event callback if profile supports it
	if emitter, ok := wsCtx.profile.(contextprofile.CompactionEventEmitter); ok {
		emitter.SetCompactionCallback(callback)
		defer emitter.SetCompactionCallback(nil)
	}

	return a.respondLoop(ctx, conv, wsCtx.states, wsCtx.tools, wsCtx.profile, callback, wsCtx.root, wsCtx.planMode)
}

func (a *Agent) respondWithCallbacks(ctx context.Context, userInput string, callback StreamCallback) (string, string, error) {
	conv := a.states.Current()
	conv.Append(state.Message{Role: "user", Content: userInput})
	if err := a.states.Save(conv); err != nil {
		return "", "", fmt.Errorf("save conversation: %w", err)
	}

	// Wire up compaction event callback if profile supports it
	if emitter, ok := a.profile.(contextprofile.CompactionEventEmitter); ok {
		emitter.SetCompactionCallback(callback)
		defer emitter.SetCompactionCallback(nil)
	}

	return a.respondLoop(ctx, conv, a.states, a.tools, a.profile, callback, "", false)
}

func (a *Agent) respondLoop(ctx context.Context, conv *state.Conversation, stateManager *state.Manager, tools *tooling.Registry, profile contextprofile.Profile, callback StreamCallback, workspaceRoot string, planMode bool) (string, string, error) {
	// Load project instructions and facts once per conversation turn
	projectInstructions := loadProjectInstructions(workspaceRoot)
	projectFacts := loadProjectFacts(workspaceRoot)

	for {
		prepared, err := profile.Prepare(ctx, conv)
		if err != nil {
			a.logger.Printf("context profile prepare failed: %v", err)
		}
		if prepared.Mutated {
			if err := stateManager.Save(conv); err != nil {
				return "", "", fmt.Errorf("save conversation: %w", err)
			}
		}
		messages := prepared.Messages
		if len(messages) == 0 {
			messages = conv.Messages()
		}

		// Inject project instructions and facts into system message
		messages = injectProjectInstructions(messages, projectInstructions)
		messages = injectProjectFacts(messages, projectFacts)

		// Inject plan mode hint if enabled
		if planMode {
			messages = injectPlanModeHint(messages)
		}

		// Inject hidden ultrathink message when force thinking is enabled
		// Only inject for user messages, not for tool call response rounds
		requestMessages := messages
		if a.cfg.ForceThinking && len(messages) > 0 && messages[len(messages)-1].Role == "user" {
			requestMessages = make([]state.Message, len(messages), len(messages)+1)
			copy(requestMessages, messages)
			requestMessages = append(requestMessages, state.Message{
				Role:    "user",
				Content: "ultrathink think very hard. reason step by step before answering.",
			})
		}

		totalChars := conversationCharCount(messages)
		a.logger.Printf("[agent] invoking provider with %d messages (~%d chars)", len(messages), totalChars)
		req := llm.ChatRequest{
			Model:       a.getActiveModel(),
			Messages:    requestMessages,
			Tools:       tools.Definitions(),
			Temperature: a.cfg.Temperature,
			Thinking: func() *llm.ThinkingOptions {
				if !a.cfg.ThinkingEnabled {
					return nil
				}
				return &llm.ThinkingOptions{Type: "enabled"}
			}(),
		}

		reqCtx, reqCancel := context.WithCancel(ctx)
		a.setInFlightCancel(reqCancel)
		resp, err := a.callProviderWithRetry(reqCtx, req, callback)
		a.clearInFlightCancel()
		reqCancel()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return "", "", nil
			}
			return "", "", fmt.Errorf("chat completion: %w", err)
		}
		logging.DevLog("received %d choices", len(resp.Choices))
		if resp.Usage != nil {
			logging.DevLog("token usage: prompt=%d completion=%d total=%d",
				resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
			a.addTokens(resp.Usage.TotalTokens)
		}
		if len(resp.Choices) == 0 {
			return "", "", fmt.Errorf("no choices returned")
		}

		choice := resp.Choices[0]
		if len(choice.Message.ToolCalls) > 0 {
			// Tool calls will be processed separately
		}
		conv.Append(choice.Message)
		if err := stateManager.Save(conv); err != nil {
			return "", "", fmt.Errorf("save conversation: %w", err)
		}

		if len(choice.Message.ToolCalls) == 0 {
			if choice.Message.Content != "" && callback != nil {
				activeProvider := a.providerCtrl.ActiveProvider()
				activeModel := a.getActiveModel()
				eventData := map[string]any{
					"content":              choice.Message.Content,
					"thinking":             choice.Message.Thinking,
					"context_chars":        conversationCharCount(conv.Messages()),
					"total_tokens":         a.getTotalTokens(),
					"context_limit_tokens": config.GetModelContextLength(activeProvider.Key, activeModel),
				}
				if resp.Usage != nil {
					eventData["usage"] = resp.Usage
				}
				callback("assistant_message", eventData)
			}
			if mutated, err := profile.AfterResponse(ctx, conv); err != nil {
				a.logger.Printf("context profile after-response failed: %v", err)
			} else if mutated {
				if err := stateManager.Save(conv); err != nil {
					return "", "", fmt.Errorf("save conversation: %w", err)
				}
				// Send updated context after AfterResponse modifies conversation
				if callback != nil {
					activeProvider := a.providerCtrl.ActiveProvider()
					activeModel := a.getActiveModel()
					callback("context_update", map[string]any{
						"context_chars":        conversationCharCount(conv.Messages()),
						"total_tokens":         a.getTotalTokens(),
						"context_limit_tokens": config.GetModelContextLength(activeProvider.Key, activeModel),
					})
				}
			}
			return choice.Message.Content, choice.FinishReason, nil
		}

		// Send assistant message with thinking/content before tool calls
		if callback != nil && (choice.Message.Thinking != "" || choice.Message.Content != "") {
			activeProvider := a.providerCtrl.ActiveProvider()
			activeModel := a.getActiveModel()
			eventData := map[string]any{
				"content":              choice.Message.Content,
				"thinking":             choice.Message.Thinking,
				"context_chars":        conversationCharCount(conv.Messages()),
				"total_tokens":         a.getTotalTokens(),
				"context_limit_tokens": config.GetModelContextLength(activeProvider.Key, activeModel),
			}
			if resp.Usage != nil {
				eventData["usage"] = resp.Usage
			}
			callback("assistant_message", eventData)
		}

		if callback != nil {
			for _, toolCall := range choice.Message.ToolCalls {
				callback("tool_call_started", map[string]any{
					"id":        toolCall.ID,
					"function":  toolCall.Function.Name,
					"arguments": toolCall.Function.Arguments,
				})
			}
		}

		if err := a.processToolCallsWithCallback(ctx, conv, choice.Message.ToolCalls, callback, stateManager, tools, planMode); err != nil {
			return "", "", err
		}
		if mutated, err := profile.AfterResponse(ctx, conv); err != nil {
			a.logger.Printf("context profile after-response failed: %v", err)
		} else if mutated {
			if err := stateManager.Save(conv); err != nil {
				return "", "", fmt.Errorf("save conversation: %w", err)
			}
			// Send updated context after AfterResponse modifies conversation
			if callback != nil {
				activeProvider := a.providerCtrl.ActiveProvider()
				activeModel := a.getActiveModel()
				callback("context_update", map[string]any{
					"context_chars":        conversationCharCount(conv.Messages()),
					"total_tokens":         a.getTotalTokens(),
					"context_limit_tokens": config.GetModelContextLength(activeProvider.Key, activeModel),
				})
			}
		}
	}
}

func (a *Agent) processToolCalls(ctx context.Context, conv *state.Conversation, calls []state.ToolCall) error {
	return a.processToolCallsWithCallback(ctx, conv, calls, nil, a.states, a.tools, false)
}

// blockedToolsInPlanMode lists tools that are not allowed when plan mode is enabled
var blockedToolsInPlanMode = map[string]bool{
	"write_file": true,
	"edit_file":  true,
}

func (a *Agent) processToolCallsWithCallback(ctx context.Context, conv *state.Conversation, calls []state.ToolCall, callback StreamCallback, stateManager *state.Manager, tools *tooling.Registry, planMode bool) error {
	for _, call := range calls {
		// Block editing tools in plan mode
		if planMode && blockedToolsInPlanMode[call.Function.Name] {
			msg := fmt.Sprintf("Tool '%s' is blocked: Plan mode is enabled. The user wants you to only analyze and plan, not make changes. Ask them to disable plan mode if they want you to implement changes.", call.Function.Name)
			logging.UserLog("plan mode: blocked %s", call.Function.Name)
			conv.Append(state.Message{Role: "tool", Name: call.Function.Name, Content: msg, ToolCallID: call.ID})
			if callback != nil {
				callback("tool_call_completed", map[string]any{
					"id":       call.ID,
					"function": call.Function.Name,
					"result":   msg,
					"error":    true,
					"blocked":  true,
				})
			}
			if err := stateManager.Save(conv); err != nil {
				return fmt.Errorf("save blocked tool result: %w", err)
			}
			continue
		}

		tool, ok := tools.Lookup(call.Function.Name)
		if !ok {
			msg := fmt.Sprintf("tool %s not registered", call.Function.Name)
			logging.ErrorLog(msg)
			conv.Append(state.Message{Role: "tool", Name: call.Function.Name, Content: msg, ToolCallID: call.ID})
			continue
		}
		var args map[string]any
		if call.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
				msg := fmt.Sprintf("invalid args for %s: %v", call.Function.Name, err)
				logging.ErrorLog(msg)
				conv.Append(state.Message{Role: "tool", Name: call.Function.Name, Content: msg, ToolCallID: call.ID})
				continue
			}
		} else {
			args = map[string]any{}
		}
		start := time.Now()
		// For recall_memory, pass conversation via context so tool can expand in-place
		// For update_plan, pass session storage path so plan is session-specific
		toolCtx := ctx
		if call.Function.Name == "recall_memory" {
			toolCtx = contextprofile.WithConversation(ctx, conv)
		} else if call.Function.Name == "update_plan" {
			toolCtx = tooling.WithSessionStorage(ctx, conv.StoragePath())
		}
		// Provide user feedback for long-running tools
		logging.UserLog("Executing tool: %s", call.Function.Name)

		result, err := tool.Call(toolCtx, args)
		if err != nil {
			result = fmt.Sprintf("tool error: %v", err)
			dur := time.Since(start).Round(time.Millisecond)
			logging.ErrorLog("tool %s failed after %s: %v", call.Function.Name, dur, err)
		} else {
			dur := time.Since(start).Round(time.Millisecond)
			originalLen := len(result)
			logging.DevLog("tool %s completed: %d bytes in %s", call.Function.Name, originalLen, dur)

			// Hard limit: truncate any tool result exceeding 50KB
			const maxToolResultSize = 50000
			if originalLen > maxToolResultSize {
				result = result[:maxToolResultSize] + fmt.Sprintf("\n\n[TRUNCATED: Tool result too large (%d chars). Showing first %d chars. Use more specific filters, smaller ranges, or pagination.]", originalLen, maxToolResultSize)
				logging.DevLog("tool %s result truncated from %d to %d bytes", call.Function.Name, originalLen, len(result))
			}
		}
		conv.Append(state.Message{Role: "tool", Name: call.Function.Name, Content: result, ToolCallID: call.ID})
		if callback != nil {
			callback("tool_call_completed", map[string]any{
				"id":            call.ID,
				"function":      call.Function.Name,
				"result":        result,
				"error":         err != nil,
				"context_chars": conversationCharCount(conv.Messages()),
				"total_tokens":  a.getTotalTokens(),
			})
		}
		if err == nil && call.Function.Name == "update_plan" {
			a.handlePlanToolResult(args, result)
			if callback != nil {
				callback("plan_update", map[string]any{
					"plan": result,
				})
			}
		}
		if err := stateManager.Save(conv); err != nil {
			return fmt.Errorf("save tool result: %w", err)
		}
	}
	return nil
}

func (a *Agent) callProviderWithRetry(ctx context.Context, req llm.ChatRequest, callback StreamCallback) (llm.ChatResponse, error) {
	const (
		maxRetries   = 5
		initialDelay = time.Second
		maxDelay     = 16 * time.Second
	)
	delay := initialDelay
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		chatCtx, chatCancel := context.WithCancel(ctx)
		start := time.Now()
		resp, err := a.client.Chat(chatCtx, req)
		elapsed := time.Since(start).Round(time.Millisecond)
		chatCancel()
		logging.DevLog("provider call finished: err=%v (attempt %d/%d, duration=%s)", err, attempt, maxRetries, elapsed)
		if err == nil {
			logging.DevLog("provider call succeeded in %s (attempt %d/%d)", elapsed, attempt, maxRetries)
			return resp, nil
		}
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			return llm.ChatResponse{}, context.Canceled
		}

		// Check if this is a structured ProviderError
		if pe, ok := llm.IsProviderError(err); ok {
			// Non-retryable errors: emit event and return immediately
			if !pe.Retryable {
				a.logger.Printf("[agent] provider error (non-retryable): %s", pe.Error())
				if callback != nil {
					callback("provider_error", buildProviderErrorPayload(pe))
				}
				return llm.ChatResponse{}, err
			}

			// Use provider-specified retry delay if available and longer than current
			if pe.RetryAfter != nil && *pe.RetryAfter > delay {
				delay = *pe.RetryAfter
			}
		}

		lastErr = err
		if attempt == maxRetries {
			break
		}
		a.logger.Printf("[agent] retrying provider call (attempt %d/%d) after %v", attempt+1, maxRetries, err)
		if callback != nil {
			callback("request_retry", map[string]any{
				"attempt":      attempt,
				"next_attempt": attempt + 1,
				"max_attempts": maxRetries,
				"delay_ms":     delay.Milliseconds(),
				"error":        err.Error(),
			})
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return llm.ChatResponse{}, context.Canceled
		case <-timer.C:
		}
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
	return llm.ChatResponse{}, lastErr
}

// buildProviderErrorPayload creates the SSE event payload for provider errors
func buildProviderErrorPayload(pe *llm.ProviderError) map[string]any {
	payload := map[string]any{
		"type":      string(pe.Type),
		"provider":  pe.Provider,
		"code":      pe.Code,
		"message":   pe.Message,
		"retryable": pe.Retryable,
	}
	if pe.ResetAt != nil {
		payload["reset_at"] = pe.ResetAt.Format("15:04:05")
		payload["reset_at_full"] = pe.ResetAt.Format(time.RFC3339)
	}
	return payload
}

func (a *Agent) setInFlightCancel(cancel context.CancelFunc) {
	a.requestCancelMu.Lock()
	a.requestCancel = cancel
	a.requestCancelMu.Unlock()
}

func (a *Agent) clearInFlightCancel() {
	a.requestCancelMu.Lock()
	a.requestCancel = nil
	a.requestCancelMu.Unlock()
}

func (a *Agent) addTokens(tokens int) {
	a.tokenMu.Lock()
	a.totalTokens += tokens
	a.tokenMu.Unlock()
}

func (a *Agent) getTotalTokens() int {
	a.tokenMu.RLock()
	defer a.tokenMu.RUnlock()
	return a.totalTokens
}

func (a *Agent) cancelInFlightRequest() bool {
	a.requestCancelMu.Lock()
	cancel := a.requestCancel
	a.requestCancel = nil
	a.requestCancelMu.Unlock()
	if cancel != nil {
		cancel()
		return true
	}
	return false
}

// CancelRequest exposes cancellation to the web UI.
func (a *Agent) CancelRequest() bool {
	return a.cancelInFlightRequest()
}

// HasInFlightRequest reports whether a provider call is still running.
func (a *Agent) HasInFlightRequest() bool {
	a.requestCancelMu.Lock()
	defer a.requestCancelMu.Unlock()
	return a.requestCancel != nil
}

func (a *Agent) ensureSessionSelected() error {
	a.sessionOnce.Do(func() {
		a.sessionOnceErr = a.initSessionSelection()
	})
	return a.sessionOnceErr
}

func (a *Agent) initSessionSelection() error {
	if key := strings.TrimSpace(a.resumeKey); key != "" {
		if _, err := a.states.Use(key); err != nil {
			logging.ErrorLog("failed to resume session %s: %v", key, err)
			return fmt.Errorf("resume session %s: %w", key, err)
		}
		logging.UserLog("Resumed session '%s'", key)
		return nil
	}
	keys := a.states.ListKeys()
	if len(keys) == 0 {
		return a.startFreshSession()
	}
	if !a.isTTY {
		fmt.Printf("Found %d existing session(s); non-interactive mode will start a new session. Use :states/:use later to switch.\n", len(keys))
		return a.startFreshSession()
	}

	fmt.Printf("Found %d stored session(s) for this project:\n", len(keys))
	for i, key := range keys {
		fmt.Printf("  %d) %s\n", i+1, key)
	}
	reader := bufio.NewReader(os.Stdin)
	loadExisting, err := promptYesNo(reader, "Load one of these sessions? [y/N]: ")
	if err != nil {
		return err
	}
	if !loadExisting {
		return a.startFreshSession()
	}

	for attempts := 0; attempts < 3; attempts++ {
		fmt.Print("Enter the session number or name to load: ")
		selection, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		key, ok := resolveSessionChoice(strings.TrimSpace(selection), keys)
		if ok {
			if _, err := a.states.Use(key); err != nil {
				return err
			}
			fmt.Printf("Loaded session '%s'.\n", key)
			return nil
		}
		fmt.Println("Invalid selection. Try again.")
	}

	fmt.Println("No valid selection provided. Starting a new session instead.")
	return a.startFreshSession()
}

func promptYesNo(reader *bufio.Reader, prompt string) (bool, error) {
	fmt.Print(prompt)
	resp, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	val := strings.ToLower(strings.TrimSpace(resp))
	return val == "y" || val == "yes", nil
}

func resolveSessionChoice(input string, keys []string) (string, bool) {
	if input == "" {
		return "", false
	}
	if idx, err := strconv.Atoi(input); err == nil {
		if idx >= 1 && idx <= len(keys) {
			return keys[idx-1], true
		}
	}
	for _, key := range keys {
		if strings.EqualFold(key, input) {
			return key, true
		}
	}
	return "", false
}

func (a *Agent) startFreshSession() error {
	base := fmt.Sprintf("session-%s", time.Now().Format("20060102-150405"))
	key := base
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			key = fmt.Sprintf("%s-%d", base, attempt+1)
		}
		if _, err := a.states.NewState(key); err == nil {
			logging.UserLog("Starting new session '%s'", key)
			return nil
		} else if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
			logging.ErrorLog("failed to create session %s: %v", key, err)
			return err
		}
	}
	fallback := fmt.Sprintf("session-%d", time.Now().UnixNano())
	if _, err := a.states.NewState(fallback); err != nil {
		logging.ErrorLog("failed to create fallback session %s: %v", fallback, err)
		return err
	}
	logging.UserLog("Starting new session '%s'", fallback)
	return nil
}

func (a *Agent) handlePlanToolResult(args map[string]any, output string) {
	action := planActionFromArgs(args)
	if action != "update" {
		return
	}
	plan, err := parsePlanSnapshot(output)
	if err != nil {
		a.logger.Printf("plan parse failed: %v", err)
		return
	}
	a.storeLastPlan(plan)
}

func (a *Agent) showPlan(ctx context.Context) error {
	plan, err := a.fetchPlanSnapshot(ctx)
	if err != nil {
		cached := a.loadLastPlan()
		if cached == nil {
			return err
		}
		fmt.Printf("Plan fetch failed (%v). Showing last known snapshot.\n", err)
		a.printPlanSnapshot("Last known plan:", cached)
		return nil
	}
	a.storeLastPlan(plan)
	a.printPlanSnapshot("Current plan:", plan)
	return nil
}

func (a *Agent) fetchPlanSnapshot(ctx context.Context) (*planSnapshot, error) {
	return fetchPlanSnapshotFromTools(ctx, a.tools)
}

func fetchPlanSnapshotFromTools(ctx context.Context, tools *tooling.Registry) (*planSnapshot, error) {
	tool, ok := tools.Lookup("update_plan")
	if !ok {
		return nil, fmt.Errorf("update_plan tool not available")
	}
	payload, err := tool.Call(ctx, map[string]any{"action": "get"})
	if err != nil {
		return nil, err
	}
	return parsePlanSnapshot(payload)
}

func (a *Agent) printPlanSnapshot(header string, plan *planSnapshot) {
	if plan == nil {
		fmt.Println("Plan is empty.")
		return
	}
	if header != "" {
		fmt.Println(header)
	}
	if !plan.UpdatedAt.IsZero() {
		fmt.Printf("  Last updated: %s\n", plan.UpdatedAt.Format(time.RFC822))
	}
	if len(plan.Steps) == 0 {
		fmt.Println("  (No plan steps recorded.)")
		return
	}
	for i, step := range plan.Steps {
		status := strings.ToUpper(strings.TrimSpace(step.Status))
		if status == "" {
			status = "PENDING"
		}
		fmt.Printf("  %d. [%s] %s\n", i+1, status, step.Step)
	}
}

func (a *Agent) storeLastPlan(plan *planSnapshot) {
	if plan == nil {
		return
	}
	clone := plan.clone()
	a.planMu.Lock()
	a.lastPlan = clone
	a.planMu.Unlock()
}

func (a *Agent) loadLastPlan() *planSnapshot {
	a.planMu.RLock()
	defer a.planMu.RUnlock()
	if a.lastPlan == nil {
		return nil
	}
	return a.lastPlan.clone()
}

type planSnapshot struct {
	UpdatedAt time.Time        `json:"updated_at"`
	Steps     []planStepRecord `json:"steps"`
}

type planStepRecord struct {
	Status string `json:"status"`
	Step   string `json:"step"`
}

func (p *planSnapshot) clone() *planSnapshot {
	if p == nil {
		return nil
	}
	clone := &planSnapshot{
		UpdatedAt: p.UpdatedAt,
		Steps:     make([]planStepRecord, len(p.Steps)),
	}
	copy(clone.Steps, p.Steps)
	return clone
}

func parsePlanSnapshot(payload string) (*planSnapshot, error) {
	var snap planSnapshot
	if err := json.Unmarshal([]byte(payload), &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

func planActionFromArgs(args map[string]any) string {
	action := "update"
	raw, ok := args["action"]
	if !ok || raw == nil {
		return action
	}
	switch v := raw.(type) {
	case string:
		val := strings.ToLower(strings.TrimSpace(v))
		if val != "" {
			return val
		}
	default:
		val := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", v)))
		if val != "" && val != "<nil>" {
			return val
		}
	}
	return action
}

func (a *Agent) handleCommand(cmd string) bool {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}
	switch parts[0] {
	case ":help":
		fmt.Println(`Commands:
  :help          show this text
  :states        list known conversation keys
  :use <key>     switch to an existing state (creates if missing)
  :new <key>     create and switch to a blank state
  :clear         wipe the current state's history
 :drop <key>    delete a stored state
 :tools         list registered tools
  :memories [n]  show up to n stored memory summaries (default 5)
  :thinking ...  toggle thinking mode (:thinking on|off)
  :reload [file] reload configuration from disk (default current config)
  :compact [n]   force compaction (ignores thresholds), protecting latest n messages (default config)
  :plan          show the most recent plan snapshot (via update_plan tool)
  :quit          exit the program`)
	case ":states":
		keys := a.states.ListKeys()
		if len(keys) == 0 {
			fmt.Println("No states yet. Use :new <name> to create one.")
			return false
		}
		fmt.Printf("States: %s\n", strings.Join(keys, ", "))
	case ":use":
		if len(parts) < 2 {
			fmt.Println(":use requires a key")
			return false
		}
		key := parts[1]
		if _, err := a.states.EnsureState(key); err != nil {
			fmt.Println(err)
			return false
		}
		fmt.Printf("Switched to %s\n", key)
	case ":new":
		if len(parts) < 2 {
			fmt.Println(":new requires a key")
			return false
		}
		key := parts[1]
		if _, err := a.states.NewState(key); err != nil {
			fmt.Println(err)
			return false
		}
		fmt.Printf("Created new state %s\n", key)
	case ":clear":
		if err := a.states.ClearCurrent(); err != nil {
			fmt.Printf("Clear failed: %v\n", err)
			return false
		}
		fmt.Println("Cleared current state.")
	case ":drop":
		if len(parts) < 2 {
			fmt.Println(":drop requires a key")
			return false
		}
		key := parts[1]
		if err := a.states.Delete(key); err != nil {
			fmt.Println(err)
			return false
		}
		fmt.Printf("Removed state %s\n", key)
	case ":tools":
		defs := a.tools.Definitions()
		if len(defs) == 0 {
			fmt.Println("No tools registered.")
			return false
		}
		fmt.Println("Tools:")
		for _, def := range defs {
			fmt.Printf("  - %s: %s\n", def.Function.Name, def.Function.Description)
		}
	case ":compact":
		setter, ok := a.profile.(contextprofile.ProtectedSetter)
		if !ok {
			fmt.Println("Current context profile does not support manual compaction.")
			return false
		}
		forcer, supportsForce := a.profile.(contextprofile.CompactionForcer)
		if !supportsForce {
			fmt.Println("Current context profile does not support forced compaction.")
			return false
		}

		target := a.cfg.ContextProtectRecent
		if len(parts) >= 2 {
			val, err := strconv.Atoi(parts[1])
			if err != nil || val < 0 {
				fmt.Println(":compact expects a non-negative integer (number of recent messages to protect).")
				return false
			}
			target = val
		}
		setter.SetProtectedRecent(target)
		defer setter.SetProtectedRecent(a.cfg.ContextProtectRecent)

		// Force compaction regardless of threshold
		forcer.ForceCompaction()

		conv := a.states.Current()
		prepared, err := a.profile.Prepare(context.Background(), conv)
		if err != nil {
			fmt.Printf("Compaction failed: %v\n", err)
			return false
		}
		if prepared.Mutated {
			conv.ReplaceMessages(prepared.Messages)
			if err := a.states.Save(conv); err != nil {
				fmt.Printf("Failed to persist conversation: %v\n", err)
				return false
			}
			fmt.Printf("Compaction run completed (protected %d most recent messages).\n", target)
		} else {
			fmt.Println("Compaction executed, but no messages qualified for summarization.")
		}
	case ":plan":
		if err := a.showPlan(context.Background()); err != nil {
			fmt.Printf("Plan fetch failed: %v\n", err)
		}
	case ":memories":
		inspector, ok := a.profile.(contextprofile.MemoryInspector)
		if !ok {
			fmt.Println("Current context profile does not expose memory details.")
			return false
		}
		limit := 5
		if len(parts) >= 2 {
			val, err := strconv.Atoi(parts[1])
			if err != nil || val <= 0 {
				fmt.Println(":memories expects a positive integer limit (e.g. :memories 5).")
				return false
			}
			limit = val
		}
		summary, err := inspector.MemorySummary(limit)
		if err != nil {
			fmt.Printf("Memory summary failed: %v\n", err)
			return false
		}
		fmt.Printf("Memories: %d total (%d pinned)\n", summary.Total, summary.Pinned)
		if len(summary.Entries) == 0 {
			fmt.Println("No stored memories.")
			return false
		}
		for _, entry := range summary.Entries {
			flag := ""
			if entry.Pinned {
				flag = " [PINNED]"
			}
			fmt.Printf("- %s%s | last access %s | %s\n", entry.ID, flag, entry.LastAccess.Format(time.RFC822), entry.Summary)
		}
	case ":thinking":
		if len(parts) == 1 {
			state := "off"
			if a.cfg.ThinkingEnabled {
				state = "on"
			}
			fmt.Printf("Thinking is %s\n", state)
			return false
		}
		switch strings.ToLower(parts[1]) {
		case "on":
			a.cfg.ThinkingEnabled = true
			if err := config.Save(a.cfg); err != nil {
				fmt.Printf("Failed to save config: %v\n", err)
			}
			fmt.Println("Thinking enabled.")
		case "off":
			a.cfg.ThinkingEnabled = false
			if err := config.Save(a.cfg); err != nil {
				fmt.Printf("Failed to save config: %v\n", err)
			}
			fmt.Println("Thinking disabled.")
		default:
			fmt.Println("Usage: :thinking on|off")
			return false
		}
	case ":reload":
		path := a.cfgPath
		if len(parts) >= 2 {
			path = parts[1]
		}
		if strings.TrimSpace(path) == "" {
			fmt.Println(":reload requires a config file path when no default is set.")
			return false
		}
		if err := a.reloadConfig(path); err != nil {
			fmt.Printf("Reload failed: %v\n", err)
		}
	case ":quit", ":exit":
		fmt.Println("Exiting per user request.")
		return true
	default:
		fmt.Printf("Unknown command %s. Try :help\n", parts[0])
	}
	return false
}

func trimLineEnding(s string) string {
	s = strings.TrimSuffix(s, "\r\n")
	s = strings.TrimSuffix(s, "\n")
	s = strings.TrimSuffix(s, "\r")
	return s
}

func (a *Agent) printResponse(text string) {
	if a.render == nil || strings.TrimSpace(text) == "" {
		fmt.Printf("%s\n", text)
		return
	}
	rendered, err := a.render.Render(text)
	if err != nil {
		a.logger.Printf("markdown render failed: %v", err)
		fmt.Printf("%s\n", text)
		return
	}
	fmt.Print(strings.TrimRight(rendered, "\n") + "\n")
}

func conversationCharCount(messages []state.Message) int {
	// Marshal messages to JSON for accurate size measurement
	msgData, err := json.Marshal(messages)
	if err != nil {
		// Fallback to manual counting
		total := 0
		for _, msg := range messages {
			total += len(msg.Content)
			for _, tool := range msg.ToolCalls {
				total += len(tool.Function.Arguments)
			}
		}
		return total
	}
	return len(msgData)
}

func (a *Agent) getActiveModel() string {
	if a.providerCtrl != nil {
		if opt := a.providerCtrl.ActiveProvider(); opt.Model != "" {
			return opt.Model
		}
	}
	return a.cfg.Model
}

func (a *Agent) ProviderOptions() []ProviderOption {
	if a.providerCtrl == nil {
		return nil
	}
	return a.providerCtrl.ProviderOptions()
}

func (a *Agent) ActiveProviderKey() string {
	if a.providerCtrl == nil {
		return ""
	}
	return a.providerCtrl.ActiveProvider().Key
}

func (a *Agent) UpdateSystemPrompt(userPrompt string) {
	// Extract only the user's portion, stripping base prompt if present
	userOnly := prompts.ExtractUserPortion(userPrompt)
	a.cfg.SystemPrompt = userOnly
	combined := prompts.Combine(userOnly)
	a.systemPrompt = combined
	if a.states != nil {
		a.states.SetSystemPrompt(combined)
	}
	a.workspacesMu.Lock()
	for _, ctx := range a.workspaceContexts {
		ctx.states.SetSystemPrompt(combined)
	}
	a.workspacesMu.Unlock()
}

func (a *Agent) SetActiveProvider(key string) error {
	if a.providerCtrl == nil {
		return fmt.Errorf("active provider cannot be changed")
	}
	if err := a.providerCtrl.SetActiveProvider(key); err != nil {
		return err
	}
	if opt := a.providerCtrl.ActiveProvider(); opt.Model != "" {
		a.cfg.Model = opt.Model
	}
	return nil
}

// ReloadProviders rebuilds the provider client from current credentials
func (a *Agent) ReloadProviders() error {
	if a.providerBuilders == nil || len(a.providerBuilders) == 0 {
		return fmt.Errorf("no provider builders available")
	}

	// Load current credentials
	creds, err := a.credManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	// Build provider registrations
	var providerRegs []ProviderRegistration
	for providerKey, builder := range a.providerBuilders {
		if creds.IsConfigured(providerKey) {
			apiKey := creds.GetAPIKey(providerKey)
			reg, err := builder(a.cfg, apiKey, a.logger)
			if err != nil {
				a.logger.Printf("Warning: %s provider init failed: %v", providerKey, err)
				continue
			}
			if reg != nil {
				providerRegs = append(providerRegs, *reg)
			}
		}
	}

	if len(providerRegs) == 0 {
		return fmt.Errorf("no providers configured")
	}

	// Determine active provider
	activeProvider := creds.DefaultProvider
	if activeProvider == "" && len(providerRegs) > 0 {
		activeProvider = providerRegs[0].Option.Key
	}

	// Create new multi-provider client
	multi, err := NewMultiProviderClient(activeProvider, providerRegs)
	if err != nil {
		return fmt.Errorf("failed to create multi-provider client: %w", err)
	}

	// Replace client and provider controller
	a.client = multi
	a.providerCtrl = providerCtrlForClient(multi)

	// Update model in config if active provider has one
	if opt := a.providerCtrl.ActiveProvider(); opt.Model != "" {
		a.cfg.Model = opt.Model
	}

	a.logger.Printf("Providers reloaded: %d configured", len(providerRegs))
	return nil
}

// SwitchWorkspace changes the active workspace by reinitializing state and tooling
func (a *Agent) SwitchWorkspace(newRoot string) error {
	// Cancel any in-flight request
	a.requestCancelMu.Lock()
	if a.requestCancel != nil {
		a.requestCancel()
		a.requestCancel = nil
	}
	a.requestCancelMu.Unlock()

	// Resolve absolute path
	absRoot, err := filepath.Abs(newRoot)
	if err != nil {
		return fmt.Errorf("resolve workspace root: %w", err)
	}

	// Verify path exists
	if _, err := os.Stat(absRoot); err != nil {
		return fmt.Errorf("workspace path does not exist: %w", err)
	}

	// Compute new data root
	dataRoot, err := ProjectStorageRoot(absRoot)
	if err != nil {
		return fmt.Errorf("compute storage root: %w", err)
	}

	// Ensure data root exists
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		return fmt.Errorf("create data root: %w", err)
	}

	// Create new conversation directory
	conversationDir := filepath.Join(dataRoot, "conversations")

	// Create new state manager
	newStates, err := state.NewManager(a.systemPrompt, conversationDir, a.logger)
	if err != nil {
		return fmt.Errorf("create state manager: %w", err)
	}

	// Update tooling options with new workspace-specific paths
	newToolOpts := a.toolOpts
	newToolOpts.WorkspaceRoot = absRoot
	newToolOpts.PlanPath = filepath.Join(dataRoot, "plan.json")
	newToolOpts.ProcessDir = filepath.Join(dataRoot, "processes")

	// Create new tooling registry
	newTools := tooling.NewRegistry(tooling.DefaultTools(newToolOpts)...)

	// Atomically swap state and tools
	a.states = newStates
	a.tools = newTools
	a.workspaceRoot = absRoot
	a.toolOpts = newToolOpts

	// Clear last plan since it's from old workspace
	a.planMu.Lock()
	a.lastPlan = nil
	a.planMu.Unlock()

	a.logger.Printf("Switched workspace to: %s (storage: %s)", absRoot, dataRoot)
	return nil
}

// GetOrCreateWorkspaceContext retrieves or creates a workspace context for the given path
func (a *Agent) GetOrCreateWorkspaceContext(workspacePath string) (*WorkspaceContext, error) {
	// Resolve absolute path
	absRoot, err := filepath.Abs(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}

	// Check cache first (read lock)
	a.workspacesMu.RLock()
	if ctx, exists := a.workspaceContexts[absRoot]; exists {
		a.workspacesMu.RUnlock()
		return ctx, nil
	}
	a.workspacesMu.RUnlock()

	// Not in cache, create new context (write lock)
	a.workspacesMu.Lock()
	defer a.workspacesMu.Unlock()

	// Double-check after acquiring write lock
	if ctx, exists := a.workspaceContexts[absRoot]; exists {
		return ctx, nil
	}

	// Verify path exists
	if _, err := os.Stat(absRoot); err != nil {
		return nil, fmt.Errorf("workspace path does not exist: %w", err)
	}

	// Compute data root
	dataRoot, err := ProjectStorageRoot(absRoot)
	if err != nil {
		return nil, fmt.Errorf("compute storage root: %w", err)
	}

	// Ensure data root exists
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create data root: %w", err)
	}

	// Create conversation directory
	conversationDir := filepath.Join(dataRoot, "conversations")

	// Create state manager
	newStates, err := state.NewManager(a.systemPrompt, conversationDir, a.logger)
	if err != nil {
		return nil, fmt.Errorf("create state manager: %w", err)
	}

	// Create tooling options
	newToolOpts := a.toolOpts
	newToolOpts.WorkspaceRoot = absRoot
	newToolOpts.PlanPath = filepath.Join(dataRoot, "plan.json")
	newToolOpts.ProcessDir = filepath.Join(dataRoot, "processes")

	// Create tooling registry
	newTools := tooling.NewRegistry(tooling.DefaultTools(newToolOpts)...)

	// Create workspace-specific config with correct memory store path
	workspaceCfg := a.cfg
	workspaceCfg.MemoryStorePath = filepath.Join(dataRoot, "memory.db")
	workspaceCfg.ConversationDir = conversationDir

	// Create workspace-specific profile
	profileType := a.cfg.ContextProfile
	// Check if client exists (avoid creating memory profile without credentials)
	if a.client == nil {
		profileType = "default"
	}
	workspaceProfile, err := contextprofile.New(profileType, contextprofile.Dependencies{
		Client:   a.client,
		Logger:   a.logger,
		Config:   workspaceCfg,
		Provider: a.activeProvider,
		Model:    a.profileModel,
	})
	if err != nil {
		return nil, fmt.Errorf("create workspace profile: %w", err)
	}

	// Register facts extractor with the profile if it supports it
	if setter, ok := workspaceProfile.(contextprofile.FactsExtractorSetter); ok {
		setter.SetFactsExtractor(&projectFactsExtractor{
			client:        a.client,
			model:         a.profileModel,
			workspaceRoot: absRoot,
			logger:        a.logger,
		})
	}

	// Add profile tools to registry
	allTools := append(tooling.DefaultTools(newToolOpts), workspaceProfile.Tools()...)
	newTools = tooling.NewRegistry(allTools...)

	// Set tool definitions in profile for compaction calculations
	if setter, ok := workspaceProfile.(interface {
		SetToolDefinitions([]tooling.ToolDefinition)
	}); ok {
		setter.SetToolDefinitions(newTools.Definitions())
	}

	// Create and cache context
	ctx := &WorkspaceContext{
		states:  newStates,
		tools:   newTools,
		profile: workspaceProfile,
		root:    absRoot,
	}
	a.workspaceContexts[absRoot] = ctx

	a.logger.Printf("Created workspace context: %s (storage: %s)", absRoot, dataRoot)
	return ctx, nil
}
