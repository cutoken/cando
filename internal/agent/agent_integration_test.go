package agent

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"cando/internal/config"
	"cando/internal/contextprofile"
	"cando/internal/credentials"
	"cando/internal/llm"
	"cando/internal/prompts"
	"cando/internal/state"
	"cando/internal/tooling"
)

type scriptedClient struct {
	mu        sync.Mutex
	responses []llm.ChatResponse
	responder func(llm.ChatRequest) llm.ChatResponse
	callCount int
}

func newScriptedClient(resps ...llm.ChatResponse) *scriptedClient {
	return &scriptedClient{
		responses: resps,
	}
}

func (c *scriptedClient) Chat(_ context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.callCount++
	if len(c.responses) > 0 {
		resp := c.responses[0]
		c.responses = c.responses[1:]
		return resp, nil
	}
	if c.responder != nil {
		return c.responder(req), nil
	}
	return llm.ChatResponse{
		Choices: []llm.ChatChoice{
			{Message: state.Message{Role: "assistant", Content: "noop"}, FinishReason: "stop"},
		},
	}, nil
}

type noopCredManager struct{}

func (noopCredManager) Load() (*credentials.Credentials, error) {
	return &credentials.Credentials{}, nil
}
func (noopCredManager) Save(*credentials.Credentials) error { return nil }
func (noopCredManager) Path() string                        { return "" }

func baseTestConfig(workspace string) config.Config {
	return config.Config{
		Model:                 "mock-model",
		Provider:              "mock",
		SystemPrompt:          "",
		Temperature:           0.1,
		ContextProfile:        "default",
		ContextMessagePercent: 0.5,
		ContextTotalPercent:   0.5,
		SummaryModel:          "mock-summary",
		WorkspaceRoot:         workspace,
		ConversationDir:       filepath.Join(workspace, "conversations"),
		MemoryStorePath:       filepath.Join(workspace, "memory.db"),
		HistoryPath:           filepath.Join(workspace, "history"),
	}
}

func newTestAgent(t *testing.T, client llm.Client, cfg config.Config) *Agent {
	t.Helper()
	logger := log.New(io.Discard, "", 0)
	if err := ensureDir(cfg.WorkspaceRoot); err != nil {
		t.Fatalf("workspace dir: %v", err)
	}
	if err := ensureDir(cfg.ConversationDir); err != nil {
		t.Fatalf("conversation dir: %v", err)
	}
	if err := ensureDir(filepath.Dir(cfg.MemoryStorePath)); err != nil {
		t.Fatalf("memory dir: %v", err)
	}

	states, err := state.NewManager(prompts.Combine(cfg.SystemPrompt), cfg.ConversationDir, logger)
	if err != nil {
		t.Fatalf("state manager: %v", err)
	}
	profile, err := contextprofile.New(cfg.ContextProfile, contextprofile.Dependencies{
		Client:   client,
		Logger:   logger,
		Config:   cfg,
		Provider: cfg.Provider,
		Model:    cfg.Model,
	})
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	toolOpts := tooling.Options{
		WorkspaceRoot: cfg.WorkspaceRoot,
		ShellTimeout:  time.Second,
		PlanPath:      filepath.Join(cfg.WorkspaceRoot, "plan.json"),
		BinDir:        filepath.Join(cfg.WorkspaceRoot, "bin"),
		ExternalData:  true,
		ProcessDir:    filepath.Join(cfg.WorkspaceRoot, "processes"),
	}
	tools := tooling.NewRegistry(tooling.DefaultTools(toolOpts)...)

	agent := New(client, cfg, "", states, profile, tools, logger, noopCredManager{}, Options{
		WorkspaceRoot: cfg.WorkspaceRoot,
	}, toolOpts)
	return agent
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func TestAgentCompactionWithMockClient(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	cfg := baseTestConfig(workspace)
	cfg.ContextProfile = "memory"
	cfg.ContextTotalPercent = 0.00001
	cfg.ContextMessagePercent = 0.00001
	client := newScriptedClient(llm.ChatResponse{
		Choices: []llm.ChatChoice{
			{Message: state.Message{Role: "assistant", Content: "done"}, FinishReason: "stop"},
		},
	})
	agent := newTestAgent(t, client, cfg)

	largePrompt := strings.Repeat("A", 5000)
	if err := agent.RunOneShot(context.Background(), largePrompt); err != nil {
		t.Fatalf("run oneshot: %v", err)
	}
	emitter, ok := agent.profile.(contextprofile.CompactionEventEmitter)
	if !ok {
		t.Fatalf("profile does not expose compaction emitter")
	}
	history := emitter.GetCompactionHistory()
	if len(history) == 0 {
		t.Fatalf("expected compaction history, got none")
	}
	// Note: With a single small assistant response ("done"), there may not be
	// enough content to compact. The test verifies compaction logic runs and
	// records history, not that messages are necessarily compacted.
	t.Logf("compaction history: %+v", history)
}

func TestAgentToolExecution(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	cfg := baseTestConfig(workspace)
	first := llm.ChatResponse{
		Choices: []llm.ChatChoice{
			{
				Message: state.Message{
					Role: "assistant",
					ToolCalls: []state.ToolCall{
						{
							ID:   "call-1",
							Type: "function",
							Function: state.FunctionCall{
								Name:      "list_directory",
								Arguments: `{"path":""}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
	}
	second := llm.ChatResponse{
		Choices: []llm.ChatChoice{
			{Message: state.Message{Role: "assistant", Content: "listed files"}, FinishReason: "stop"},
		},
	}
	client := newScriptedClient(first, second)
	agent := newTestAgent(t, client, cfg)
	if err := agent.RunOneShot(context.Background(), "list files"); err != nil {
		t.Fatalf("run oneshot: %v", err)
	}
	messages := agent.states.Current().Messages()
	found := false
	for _, msg := range messages {
		if msg.Role == "tool" && msg.Name == "list_directory" {
			if msg.Content == "" {
				t.Fatalf("list_directory tool returned empty content")
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected tool output in conversation log")
	}
}

func TestAgentToolFailure(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	cfg := baseTestConfig(workspace)
	first := llm.ChatResponse{
		Choices: []llm.ChatChoice{
			{
				Message: state.Message{
					Role: "assistant",
					ToolCalls: []state.ToolCall{
						{
							ID:   "call-err",
							Type: "function",
							Function: state.FunctionCall{
								Name:      "read_file",
								Arguments: `{"path":"missing.txt"}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
	}
	second := llm.ChatResponse{
		Choices: []llm.ChatChoice{
			{Message: state.Message{Role: "assistant", Content: "failed"}, FinishReason: "stop"},
		},
	}
	client := newScriptedClient(first, second)
	agent := newTestAgent(t, client, cfg)
	if err := agent.RunOneShot(context.Background(), "read file"); err != nil {
		t.Fatalf("run oneshot: %v", err)
	}
	var toolMsg *state.Message
	for _, msg := range agent.states.Current().Messages() {
		if msg.Role == "tool" && msg.Name == "read_file" {
			toolMsg = &msg
			break
		}
	}
	if toolMsg == nil || !strings.Contains(toolMsg.Content, "tool error") {
		t.Fatalf("expected tool error message, got %+v", toolMsg)
	}
}

func TestAgentProviderSwitching(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	cfg := baseTestConfig(workspace)

	alpha := &scriptedClient{
		responder: func(req llm.ChatRequest) llm.ChatResponse {
			last := req.Messages[len(req.Messages)-1].Content
			return llm.ChatResponse{
				Choices: []llm.ChatChoice{
					{Message: state.Message{Role: "assistant", Content: "alpha:" + last}, FinishReason: "stop"},
				},
			}
		},
	}
	beta := &scriptedClient{
		responder: func(req llm.ChatRequest) llm.ChatResponse {
			last := req.Messages[len(req.Messages)-1].Content
			return llm.ChatResponse{
				Choices: []llm.ChatChoice{
					{Message: state.Message{Role: "assistant", Content: "beta:" + last}, FinishReason: "stop"},
				},
			}
		},
	}
	regs := []ProviderRegistration{
		{
			Option: ProviderOption{Key: "alpha", Label: "Alpha", Model: "model-alpha"},
			Client: alpha,
		},
		{
			Option: ProviderOption{Key: "beta", Label: "Beta", Model: "model-beta"},
			Client: beta,
		},
	}
	multi, err := NewMultiProviderClient("alpha", regs)
	if err != nil {
		t.Fatalf("multi provider: %v", err)
	}
	cfg.Provider = "alpha"
	cfg.Model = "model-alpha"
	agent := newTestAgent(t, multi, cfg)

	if err := agent.RunOneShot(context.Background(), "hello"); err != nil {
		t.Fatalf("run oneshot alpha: %v", err)
	}
	msgs := agent.states.Current().Messages()
	if last := msgs[len(msgs)-1].Content; !strings.Contains(last, "alpha:hello") {
		t.Fatalf("expected alpha response, got %s", last)
	}

	if err := agent.SetActiveProvider("beta"); err != nil {
		t.Fatalf("set provider: %v", err)
	}
	if agent.ActiveProviderKey() != "beta" {
		t.Fatalf("Active provider not updated; got %s", agent.ActiveProviderKey())
	}
	if agent.cfg.Model != "model-beta" {
		t.Fatalf("config model not updated; got %s", agent.cfg.Model)
	}

	if err := agent.RunOneShot(context.Background(), "world"); err != nil {
		t.Fatalf("run oneshot beta: %v", err)
	}
	msgs = agent.states.Current().Messages()
	if last := msgs[len(msgs)-1].Content; !strings.Contains(last, "beta:world") {
		t.Fatalf("expected beta response, got %s", last)
	}
}
