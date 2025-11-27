package agent

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cando/internal/config"
	"cando/internal/contextprofile"
	"cando/internal/credentials"
	"cando/internal/state"
	"cando/internal/tooling"
)

//go:embed webui/*.tmpl
var webTemplates embed.FS

//go:embed webui/styles.css
var webStyles []byte

//go:embed webui/app.js
var webScript []byte

//go:embed webui/alpine.min.js
var alpineScript []byte

//go:embed webui/lucide.min.js
var lucideScript []byte

//go:embed webui/openrouter-models.json
var openrouterModels []byte

var templates *template.Template

// RunWeb launches the embedded HTML interface instead of the CLI REPL.
func (a *Agent) RunWeb(ctx context.Context, addr string) error {
	clean := strings.TrimSpace(addr)
	if clean == "" {
		clean = "127.0.0.1:3737"
	}
	server := &webServer{
		agent:  a,
		addr:   clean,
		logger: a.logger,
	}
	return server.run(ctx)
}

type webServer struct {
	agent            *Agent
	addr             string
	logger           *log.Logger
	actualAddr       string
	workspaceManager *WorkspaceManager
}

func (s *webServer) run(ctx context.Context) error {
	if s.logger == nil {
		s.logger = log.Default()
	}

	// Initialize workspace manager
	wsMgr, err := NewWorkspaceManager()
	if err != nil {
		return fmt.Errorf("failed to init workspace manager: %w", err)
	}
	s.workspaceManager = wsMgr

	// Load templates on startup
	if err := loadTemplates(); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	actualAddr := listener.Addr().String()
	s.actualAddr = actualAddr
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/sessions", s.handleSessionsPage)
	mux.HandleFunc("/app.css", s.handleStyles)
	mux.HandleFunc("/app.js", s.handleScript)
	mux.HandleFunc("/alpine.js", s.handleAlpine)
	mux.HandleFunc("/lucide.js", s.handleLucide)
	mux.HandleFunc("/openrouter-models.json", s.handleOpenRouterModels)
	mux.HandleFunc("/api/session", s.handleSession)
	mux.HandleFunc("/api/prompt", s.handlePrompt)
	mux.HandleFunc("/api/stream", s.handleStream)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/thinking", s.handleThinking)
	mux.HandleFunc("/api/force-thinking", s.handleForceThinking)
	mux.HandleFunc("/api/system-prompt", s.handleSystemPrompt)
	mux.HandleFunc("/api/cancel", s.handleCancel)
	mux.HandleFunc("/api/provider", s.handleProviderSwitch)
	mux.HandleFunc("/api/provider/model", s.handleProviderModelUpdate)
	mux.HandleFunc("/api/compaction-history", s.handleCompactionHistory)
	mux.HandleFunc("/api/credentials", s.handleCredentials)
	mux.HandleFunc("/api/files", s.handleFileSearch)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/workspaces", s.handleWorkspaces)
	mux.HandleFunc("/api/workspace/add", s.handleWorkspaceAdd)
	mux.HandleFunc("/api/workspace/switch", s.handleWorkspaceSwitch)
	mux.HandleFunc("/api/workspace/remove", s.handleWorkspaceRemove)
	mux.HandleFunc("/api/browse", s.handleBrowse)
	mux.HandleFunc("/api/folder/create", s.handleFolderCreate)
	mux.HandleFunc("/api/branch", s.handleBranch)

	server := &http.Server{
		Addr:    actualAddr,
		Handler: s.logRequests(mux),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	fmt.Printf("Cando web UI listening at http://%s\n", actualAddr)
	s.logger.Printf("web UI listening on http://%s\n", actualAddr)
	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *webServer) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Printf("[%s] %s %s (%s)", r.RemoteAddr, r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func (s *webServer) logRequestError(r *http.Request, status int, message string) {
	if s.logger == nil {
		return
	}
	workspace := s.getWorkspaceFromRequest(r)
	s.logger.Printf("[WEB] error status=%d method=%s path=%s workspace=%s remote=%s: %s",
		status, r.Method, r.URL.Path, workspace, r.RemoteAddr, message)
}

func (s *webServer) respondError(w http.ResponseWriter, r *http.Request, status int, message string) {
	s.logRequestError(r, status, message)
	http.Error(w, message, status)
}

func loadTemplates() error {
	devMode := os.Getenv("DEV_MODE") == "true"

	if devMode {
		// Development: load from disk for hot reload
		tmpl, err := template.ParseGlob("internal/agent/webui/*.tmpl")
		if err != nil {
			return fmt.Errorf("failed to parse templates from disk: %w", err)
		}
		templates = tmpl
	} else {
		// Production: use embedded templates
		tmpl, err := template.ParseFS(webTemplates, "webui/*.tmpl")
		if err != nil {
			return fmt.Errorf("failed to parse embedded templates: %w", err)
		}
		templates = tmpl
	}

	return nil
}

func (s *webServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Reload templates in dev mode for hot reload
	if os.Getenv("DEV_MODE") == "true" {
		if err := loadTemplates(); err != nil {
			s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("Template error: %v", err))
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "main.tmpl", nil); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("Template execution error: %v", err))
		return
	}
}

func (s *webServer) handleSessionsPage(w http.ResponseWriter, r *http.Request) {
	// Reload templates in dev mode for hot reload
	if os.Getenv("DEV_MODE") == "true" {
		if err := loadTemplates(); err != nil {
			s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("Template error: %v", err))
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "sessions.tmpl", nil); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("Template execution error: %v", err))
		return
	}
}

func (s *webServer) handleStyles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write(webStyles)
}

func (s *webServer) handleScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(webScript)
}

func (s *webServer) handleAlpine(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(alpineScript)
}

func (s *webServer) handleLucide(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(lucideScript)
}

func (s *webServer) handleOpenRouterModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(openrouterModels)
}

func (s *webServer) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.writeSessionPayload(w, r)
}

func (s *webServer) handlePrompt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid payload")
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		s.respondError(w, r, http.StatusBadRequest, "content is required")
		return
	}
	workspace := s.getWorkspaceFromRequest(r)
	if workspace == "" || !s.workspaceExists(workspace) {
		s.respondError(w, r, http.StatusBadRequest, "select a workspace first")
		return
	}
	wsCtx, err := s.agent.GetOrCreateWorkspaceContext(workspace)
	if err != nil {
		s.respondError(w, r, http.StatusBadRequest, fmt.Sprintf("get workspace context: %v", err))
		return
	}
	if s.agent.HasInFlightRequest() {
		s.respondError(w, r, http.StatusConflict, "another request is already running")
		return
	}
	if _, _, err := s.agent.respondWithCallbacksForWorkspace(r.Context(), content, nil, wsCtx); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("request failed: %v", err))
		return
	}
	s.writeSessionPayload(w, r)
}

func (s *webServer) handleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid payload")
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		s.respondError(w, r, http.StatusBadRequest, "content is required")
		return
	}
	if s.agent.HasInFlightRequest() {
		s.respondError(w, r, http.StatusConflict, "another request is already running")
		return
	}

	// Get workspace context for current workspace
	workspace := s.getWorkspaceFromRequest(r)
	if workspace == "" || !s.workspaceExists(workspace) {
		s.respondError(w, r, http.StatusBadRequest, "select a workspace first")
		return
	}
	s.agent.logger.Printf("[WEB] handleStream: workspace=%s", workspace)
	wsCtx, err := s.agent.GetOrCreateWorkspaceContext(workspace)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("get workspace context: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.respondError(w, r, http.StatusInternalServerError, "streaming not supported")
		return
	}

	sendEvent := func(eventType string, data any) error {
		payload, err := json.Marshal(map[string]any{
			"type": eventType,
			"data": data,
		})
		if err != nil {
			s.logRequestError(r, http.StatusInternalServerError, fmt.Sprintf("stream marshal %s event failed: %v", eventType, err))
			return err
		}
		_, err = fmt.Fprintf(w, "data: %s\n\n", string(payload))
		if err != nil {
			s.logRequestError(r, http.StatusInternalServerError, fmt.Sprintf("stream write %s event failed: %v", eventType, err))
			return err
		}
		flusher.Flush()
		return nil
	}

	// Handle :compact command
	if strings.HasPrefix(content, ":compact") {
		if err := s.handleCompactCommand(r.Context(), content, wsCtx, sendEvent); err != nil {
			s.logRequestError(r, http.StatusInternalServerError, fmt.Sprintf("compact command failed: %v", err))
			sendEvent("error", map[string]string{"message": err.Error()})
			return
		}
		sendEvent("complete", map[string]string{"status": "done"})
		return
	}

	if _, _, err := s.agent.respondWithCallbacksForWorkspace(r.Context(), content, sendEvent, wsCtx); err != nil {
		s.logRequestError(r, http.StatusInternalServerError, fmt.Sprintf("stream request failed: %v", err))
		sendEvent("error", map[string]string{"message": err.Error()})
		return
	}

	sendEvent("complete", map[string]string{"status": "done"})
}

func (s *webServer) handleCompactCommand(ctx context.Context, content string, wsCtx *WorkspaceContext, sendEvent func(string, any) error) error {
	// Parse command: ":compact" or ":compact <n>"
	parts := strings.Fields(content)

	// Get profile and check support (use workspace-specific profile)
	setter, ok := wsCtx.profile.(contextprofile.ProtectedSetter)
	if !ok {
		return fmt.Errorf("current context profile does not support manual compaction")
	}
	forcer, supportsForce := wsCtx.profile.(contextprofile.CompactionForcer)
	if !supportsForce {
		return fmt.Errorf("current context profile does not support forced compaction")
	}

	// Parse protected message count
	target := s.agent.cfg.ContextProtectRecent
	if len(parts) >= 2 {
		var val int
		if _, err := fmt.Sscanf(parts[1], "%d", &val); err != nil || val < 0 {
			return fmt.Errorf(":compact expects a non-negative integer (number of recent messages to protect)")
		}
		target = val
	}

	// Send status event
	sendEvent("status", map[string]any{
		"message": fmt.Sprintf("Starting forced compaction (protecting %d recent messages)...", target),
	})

	// Set protected count temporarily
	originalProtected := s.agent.cfg.ContextProtectRecent
	setter.SetProtectedRecent(target)
	defer setter.SetProtectedRecent(originalProtected)

	// Force compaction
	forcer.ForceCompaction()

	// Run compaction
	conv := wsCtx.states.Current()
	prepared, err := wsCtx.profile.Prepare(ctx, conv)
	if err != nil {
		return fmt.Errorf("compaction failed: %w", err)
	}

	if prepared.Mutated {
		conv.ReplaceMessages(prepared.Messages)
		if err := wsCtx.states.Save(conv); err != nil {
			return fmt.Errorf("failed to persist conversation: %w", err)
		}

		// Send success event
		sendEvent("assistant_message", map[string]any{
			"content": fmt.Sprintf("✓ Compaction completed successfully (protected %d most recent messages)", target),
			"role":    "assistant",
		})
	} else {
		// Send info event
		sendEvent("assistant_message", map[string]any{
			"content": "Compaction executed, but no messages qualified for summarization.",
			"role":    "assistant",
		})
	}

	return nil
}

func (s *webServer) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Action string `json:"action"`
		Key    string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid payload")
		return
	}

	// Get workspace context
	workspace := s.getWorkspaceFromRequest(r)
	if workspace == "" || !s.workspaceExists(workspace) {
		s.respondError(w, r, http.StatusBadRequest, "select a workspace first")
		return
	}
	s.agent.logger.Printf("[WEB] handleState: workspace=%s, action=%s, key=%s", workspace, req.Action, req.Key)
	wsCtx, err := s.agent.GetOrCreateWorkspaceContext(workspace)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("get workspace context: %v", err))
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	key := strings.TrimSpace(req.Key)
	switch action {
	case "switch":
		if key == "" {
			s.respondError(w, r, http.StatusBadRequest, "key is required")
			return
		}
		if _, err := wsCtx.states.EnsureState(key); err != nil {
			s.respondError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	case "new":
		if key == "" {
			s.respondError(w, r, http.StatusBadRequest, "key is required")
			return
		}
		if _, err := wsCtx.states.NewState(key); err != nil {
			s.respondError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	case "delete":
		if key == "" {
			s.respondError(w, r, http.StatusBadRequest, "key is required")
			return
		}
		if err := wsCtx.states.Delete(key); err != nil {
			s.respondError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	case "clear":
		if err := wsCtx.states.ClearCurrent(); err != nil {
			s.respondError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	default:
		s.respondError(w, r, http.StatusBadRequest, "unknown action")
		return
	}
	s.writeSessionPayload(w, r)
}

func (s *webServer) handleThinking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid payload")
		return
	}
	s.agent.thinkingEnabled = req.Enabled
	s.agent.cfg.ThinkingEnabled = req.Enabled

	// Save to disk
	if err := config.Save(s.agent.cfg); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to save config: %v", err))
		return
	}

	s.writeSessionPayload(w, r)
}

func (s *webServer) handleForceThinking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid payload")
		return
	}
	s.agent.forceThinking = req.Enabled
	s.agent.cfg.ForceThinking = req.Enabled

	// Save to disk
	if err := config.Save(s.agent.cfg); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to save config: %v", err))
		return
	}

	s.writeSessionPayload(w, r)
}

func (s *webServer) handleProviderSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.agent.providerCtrl == nil {
		s.respondError(w, r, http.StatusBadRequest, "provider switching is not available")
		return
	}
	var req struct {
		Key string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := s.agent.SetActiveProvider(req.Key); err != nil {
		s.respondError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// Persist new default provider in credentials
	if s.agent.credManager != nil {
		if creds, err := s.agent.credManager.Load(); err != nil {
			s.logger.Printf("Failed to reload credentials while switching provider: %v", err)
		} else {
			if creds.DefaultProvider != req.Key {
				creds.DefaultProvider = req.Key
				if err := s.agent.credManager.Save(creds); err != nil {
					s.logger.Printf("Failed to save credentials after provider switch: %v", err)
					s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to persist provider switch: %v", err))
					return
				}
			}
		}
	}

	s.writeSessionPayload(w, r)
}

// handleProviderModelUpdate handles all model type updates (main, summary, vision)
func (s *webServer) handleProviderModelUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Provider  string `json:"provider"`
		ModelType string `json:"model_type"` // "main", "summary", or "vision"
		Model     string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid payload")
		return
	}

	if req.Provider == "" || req.Model == "" {
		s.respondError(w, r, http.StatusBadRequest, "provider and model are required")
		return
	}

	// Default to "main" if model_type not specified (backwards compatibility)
	if req.ModelType == "" {
		req.ModelType = "main"
	}

	// Update the appropriate config field based on model type
	switch req.ModelType {
	case "main":
		if s.agent.cfg.ProviderModels == nil {
			s.agent.cfg.ProviderModels = make(map[string]string)
		}
		s.agent.cfg.ProviderModels[req.Provider] = req.Model
	case "summary":
		if s.agent.cfg.ProviderSummaryModels == nil {
			s.agent.cfg.ProviderSummaryModels = make(map[string]string)
		}
		s.agent.cfg.ProviderSummaryModels[req.Provider] = req.Model
		// Update current summary model if this is the active provider
		if req.Provider == s.agent.cfg.Provider {
			s.agent.cfg.SummaryModel = req.Model
		}
	case "vision":
		if s.agent.cfg.ProviderVLModels == nil {
			s.agent.cfg.ProviderVLModels = make(map[string]string)
		}
		s.agent.cfg.ProviderVLModels[req.Provider] = req.Model
	default:
		s.respondError(w, r, http.StatusBadRequest, "invalid model_type: must be main, summary, or vision")
		return
	}

	// Save config to disk
	if err := config.Save(s.agent.cfg); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to save config: %v", err))
		return
	}

	s.logger.Printf("Updated %s %s model to %s", req.Provider, req.ModelType, req.Model)

	// Reload providers for main model changes
	if req.ModelType == "main" {
		if err := s.agent.ReloadProviders(); err != nil {
			s.logger.Printf("Warning: failed to reload providers after model change: %v", err)
		}
	}

	s.writeJSON(w, r, map[string]any{
		"success": true,
		"message": fmt.Sprintf("%s model updated to %s!", req.ModelType, req.Model),
	})
}

func (s *webServer) handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	cancelled := s.agent.CancelRequest()
	resp := map[string]any{
		"cancelled": cancelled,
		"running":   s.agent.HasInFlightRequest(),
	}
	s.writeJSON(w, r, resp)
}

func (s *webServer) handleCompactionHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get workspace from request
	workspace := s.getWorkspaceFromRequest(r)
	if workspace == "" || !s.workspaceExists(workspace) {
		s.respondError(w, r, http.StatusBadRequest, "select a workspace first")
		return
	}

	// Get workspace context
	wsCtx, err := s.agent.GetOrCreateWorkspaceContext(workspace)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("get workspace context: %v", err))
		return
	}

	// Check workspace profile (not agent.profile)
	emitter, ok := wsCtx.profile.(contextprofile.CompactionEventEmitter)
	if !ok {
		s.writeJSON(w, r, map[string]any{
			"history": []contextprofile.CompactionEvent{},
		})
		return
	}

	history := emitter.GetCompactionHistory()
	s.writeJSON(w, r, map[string]any{
		"history": history,
	})
}

type sessionPayload struct {
	CurrentKey            string            `json:"current_key"`
	Keys                  []string          `json:"keys"`
	Sessions              []state.Summary   `json:"sessions"`
	Messages              []state.Message   `json:"messages"`
	Thinking              bool              `json:"thinking"`
	ForceThinking         bool              `json:"force_thinking"`
	SystemPrompt          string            `json:"system_prompt"`
	Running               bool              `json:"running"`
	ContextChars          int               `json:"context_chars"`
	ContextLimitTokens    int               `json:"context_limit_tokens,omitempty"`
	TotalTokens           int               `json:"total_tokens"`
	Model                 string            `json:"model"`
	SummaryModel          string            `json:"summary_model,omitempty"`
	Providers             []ProviderOption  `json:"providers,omitempty"`
	ProviderModels        map[string]string `json:"provider_models,omitempty"`
	ProviderSummaryModels map[string]string `json:"provider_summary_models,omitempty"`
	ProviderVLModels      map[string]string `json:"provider_vl_models,omitempty"`
	CurrentProvider       string            `json:"current_provider,omitempty"`
	Plan                  *planSnapshot     `json:"plan,omitempty"`
	PlanError             string            `json:"plan_error,omitempty"`
	Workdir               string            `json:"workdir,omitempty"`
	Config                *configSnapshot   `json:"config,omitempty"`
	Workspace             *Workspace        `json:"workspace,omitempty"`
	Workspaces            []Workspace       `json:"workspaces,omitempty"`
	RecentWorkspaces      []Workspace       `json:"recent_workspaces,omitempty"`
}

type configSnapshot struct {
	ContextProfile             string  `json:"context_profile"`
	ContextMessagePercent      float64 `json:"context_message_percent"`
	ContextConversationPercent float64 `json:"context_conversation_percent"`
	ContextProtectRecent       int     `json:"context_protect_recent"`
	SystemPrompt               string  `json:"system_prompt"`
}

// getProvidersFromDisk reads current credentials and config from disk to build fresh provider list
func (s *webServer) getProvidersFromDisk() ([]ProviderOption, string) {
	if s.agent.credManager == nil || s.agent.providerBuilders == nil {
		return []ProviderOption{}, ""
	}

	// Load credentials from disk
	creds, err := s.agent.credManager.Load()
	if err != nil {
		s.logger.Printf("Failed to load credentials: %v", err)
		return []ProviderOption{}, ""
	}

	// Build provider options from configured providers
	var providers []ProviderOption
	for providerKey, builder := range s.agent.providerBuilders {
		if creds.IsConfigured(providerKey) {
			apiKey := creds.GetAPIKey(providerKey)
			reg, err := builder(s.agent.cfg, apiKey, s.logger)
			if err != nil {
				continue
			}
			if reg != nil {
				providers = append(providers, reg.Option)
			}
		}
	}

	// Get current provider from credentials
	currentProvider := creds.DefaultProvider
	if currentProvider == "" && len(providers) > 0 {
		currentProvider = providers[0].Key
	}

	return providers, currentProvider
}

// getWorkspaceFromRequest extracts workspace from query param or localStorage header
func (s *webServer) getWorkspaceFromRequest(r *http.Request) string {
	// Check query parameter first
	if ws := r.URL.Query().Get("workspace"); ws != "" {
		return ws
	}
	// Check header (sent by frontend from localStorage)
	if ws := r.Header.Get("X-Workspace"); ws != "" {
		return ws
	}
	if s.workspaceManager != nil {
		if current := s.workspaceManager.Current(); current != nil {
			return current.Path
		}
	}
	return ""
}

func (s *webServer) workspaceExists(path string) bool {
	if path == "" || s.workspaceManager == nil {
		return false
	}
	return s.workspaceManager.GetByPath(path) != nil
}

func (s *webServer) writeSessionPayload(w http.ResponseWriter, r *http.Request) {
	workspace := s.getWorkspaceFromRequest(r)
	if workspace != "" && !s.workspaceExists(workspace) {
		s.logger.Printf("[WEB] requested workspace %s not found; returning empty state", workspace)
		workspace = ""
	}
	s.agent.logger.Printf("[WEB] writeSessionPayload: workspace=%s", workspace)
	payload, err := s.buildSessionPayload(r.Context(), workspace)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to build session: %v", err))
		return
	}
	s.writeJSON(w, r, payload)
}

func (s *webServer) buildSessionPayload(ctx context.Context, workspacePath string) (sessionPayload, error) {
	providers, currentProvider := s.getProvidersFromDisk()
	activeModel := s.agent.getActiveModel()

	payload := sessionPayload{
		Thinking:              s.agent.thinkingEnabled,
		ForceThinking:         s.agent.forceThinking,
		SystemPrompt:          s.agent.cfg.SystemPrompt,
		Running:               s.agent.HasInFlightRequest(),
		TotalTokens:           s.agent.getTotalTokens(),
		Model:                 activeModel,
		SummaryModel:          s.agent.cfg.SummaryModel,
		Providers:             providers,
		ProviderModels:        s.agent.cfg.ProviderModels,
		ProviderSummaryModels: s.agent.cfg.ProviderSummaryModels,
		ProviderVLModels:      s.agent.cfg.ProviderVLModels,
		CurrentProvider:       currentProvider,
	}
	if s.workspaceManager != nil {
		payload.Workspaces = s.workspaceManager.List()
		payload.RecentWorkspaces = s.workspaceManager.Recent()
	}

	if workspacePath == "" {
		return payload, nil
	}

	wsCtx, err := s.agent.GetOrCreateWorkspaceContext(workspacePath)
	if err != nil {
		return payload, fmt.Errorf("get workspace context: %w", err)
	}

	conv := wsCtx.states.Current()
	messages := conv.Messages()
	// Pass session storage path to tools for session-specific plan
	toolCtx := tooling.WithSessionStorage(ctx, conv.StoragePath())
	plan, planErr := fetchPlanSnapshotFromTools(toolCtx, wsCtx.tools)
	if planErr != nil {
		if cached := s.agent.loadLastPlan(); cached != nil {
			plan = cached
		}
	} else if plan != nil {
		s.agent.storeLastPlan(plan)
	}

	payload.CurrentKey = conv.Key()
	payload.Keys = wsCtx.states.ListKeys()
	payload.Sessions = wsCtx.states.Summaries()
	payload.Messages = filterSystemMessages(messages)
	payload.ContextChars = conversationCharCount(messages)
	payload.Plan = plan
	payload.Workdir = wsCtx.root
	payload.Config = &configSnapshot{
		ContextProfile:             s.agent.cfg.ContextProfile,
		ContextMessagePercent:      s.agent.cfg.ContextMessagePercent,
		ContextConversationPercent: s.agent.cfg.ContextTotalPercent,
		ContextProtectRecent:       s.agent.cfg.ContextProtectRecent,
		SystemPrompt:               s.agent.cfg.SystemPrompt,
	}
	if planErr != nil {
		payload.PlanError = planErr.Error()
	}

	activeProvider := s.agent.ActiveProviderKey()
	if activeProvider == "" {
		activeProvider = currentProvider
	}
	payload.ContextLimitTokens = config.GetModelContextLength(activeProvider, payload.Model)

	if s.workspaceManager != nil {
		payload.Workspace = s.workspaceManager.GetByPath(wsCtx.root)
	}

	return payload, nil
}

func filterSystemMessages(messages []state.Message) []state.Message {
	if len(messages) == 0 {
		return messages
	}
	filtered := make([]state.Message, 0, len(messages))
	for idx, msg := range messages {
		if idx == 0 && strings.EqualFold(msg.Role, "system") {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

func (s *webServer) handleCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Check if credentials exist
		if s.agent.credManager == nil {
			s.writeJSON(w, r, map[string]any{
				"configured":            false,
				"zai_configured":        false,
				"openrouter_configured": false,
			})
			return
		}

		creds, err := s.agent.credManager.Load()
		if err != nil {
			s.writeJSON(w, r, map[string]any{
				"configured":            false,
				"zai_configured":        false,
				"openrouter_configured": false,
			})
			return
		}

		configured := creds.HasAnyProvider()
		resp := map[string]any{
			"configured":            configured,
			"provider":              creds.DefaultProvider,
			"zai_configured":        creds.IsConfigured("zai"),
			"openrouter_configured": creds.IsConfigured("openrouter"),
		}
		if p, ok := creds.Providers["zai"]; ok {
			resp["zai_vision_model"] = p.VisionModel
		}
		if p, ok := creds.Providers["openrouter"]; ok {
			resp["openrouter_vision_model"] = p.VisionModel
		}
		s.writeJSON(w, r, resp)
		return
	}

	if r.Method == http.MethodPost {
		// Save credentials
		var req struct {
			Provider    string `json:"provider"`
			APIKey      string `json:"api_key"`
			VisionModel string `json:"vision_model,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		if req.Provider == "" || req.APIKey == "" {
			s.respondError(w, r, http.StatusBadRequest, "provider and api_key required")
			return
		}

		if s.agent.credManager == nil {
			s.respondError(w, r, http.StatusInternalServerError, "credential manager not available")
			return
		}

		// Load existing credentials - Load() returns empty creds if file doesn't exist
		creds, err := s.agent.credManager.Load()
		if err != nil {
			// Real error (not file-doesn't-exist) - fail instead of overwriting
			s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to load existing credentials: %v", err))
			return
		}

		// Ensure Providers map exists
		if creds.Providers == nil {
			creds.Providers = make(map[string]credentials.Provider)
		}

		// Add or update this provider (preserves other providers)
		creds.Providers[req.Provider] = credentials.Provider{
			APIKey:      req.APIKey,
			VisionModel: req.VisionModel,
		}

		// Set as default if no default provider set
		if creds.DefaultProvider == "" {
			creds.DefaultProvider = req.Provider
		}

		// Save updated credentials
		if err := s.agent.credManager.Save(creds); err != nil {
			s.respondError(w, r, http.StatusInternalServerError, err.Error())
			return
		}

		// Create default config for the provider
		if err := config.EnsureDefaultConfig(req.Provider); err != nil {
			s.logger.Printf("Warning: failed to create default config: %v", err)
		}

		// Reload providers dynamically
		if err := s.agent.ReloadProviders(); err != nil {
			s.logger.Printf("Warning: failed to reload providers: %v", err)
			s.writeJSON(w, r, map[string]any{
				"success": true,
				"message": "Credentials saved but provider reload failed. Please restart Cando.",
			})
			return
		}

		s.writeJSON(w, r, map[string]any{
			"success": true,
			"message": fmt.Sprintf("%s provider configured successfully!", strings.ToUpper(req.Provider)),
		})
		return
	}

	s.respondError(w, r, http.StatusMethodNotAllowed, "Method not allowed")
}

func (s *webServer) writeJSON(w http.ResponseWriter, r *http.Request, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, err.Error())
	}
}

func (s *webServer) handleFileSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	workspaceRoot := s.getWorkspaceFromRequest(r)
	if workspaceRoot != "" && !s.workspaceExists(workspaceRoot) {
		workspaceRoot = ""
	}
	if workspaceRoot == "" {
		workspaceRoot = s.agent.workspaceRoot
	}
	if workspaceRoot == "" {
		workspaceRoot = "."
	}
	if abs, err := filepath.Abs(workspaceRoot); err == nil {
		workspaceRoot = abs
	}

	// Common directories to skip
	skipDirs := map[string]bool{
		"node_modules": true,
		".git":         true,
		"dist":         true,
		"build":        true,
		"out":          true,
		"target":       true,
		"vendor":       true,
		"bin":          true,
		"obj":          true,
		".next":        true,
		".nuxt":        true,
		"coverage":     true,
	}

	type match struct {
		name  string
		path  string
		typ   string
		depth int
	}

	var matches []match
	maxCollect := 500 // Safety limit to prevent memory issues

	err := filepath.Walk(workspaceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}

		base := filepath.Base(path)

		// Skip hidden files and directories
		if strings.HasPrefix(base, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip common build/dependency directories
		if info.IsDir() && skipDirs[base] {
			return filepath.SkipDir
		}

		// Get relative path from workspace root
		relPath, err := filepath.Rel(workspaceRoot, path)
		if err != nil {
			relPath = path
		}

		// Match all files when query is empty, otherwise do substring match
		matched := query == "" ||
			strings.Contains(strings.ToLower(base), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(relPath), strings.ToLower(query))

		if matched {
			// Calculate depth (number of path separators)
			depth := strings.Count(relPath, string(filepath.Separator))

			fileType := "file"
			if info.IsDir() {
				fileType = "dir"
			}

			matches = append(matches, match{
				name:  base,
				path:  relPath,
				typ:   fileType,
				depth: depth,
			})

			// Safety limit to prevent memory issues in huge repos
			if len(matches) >= maxCollect {
				return filepath.SkipAll
			}
		}

		return nil
	})

	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Sort by depth (root files first), then alphabetically
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].depth != matches[j].depth {
			return matches[i].depth < matches[j].depth
		}
		return matches[i].path < matches[j].path
	})

	// Apply limit after sorting
	limit := 50
	if len(matches) > limit {
		matches = matches[:limit]
	}

	// Convert to response format
	result := make([]map[string]string, len(matches))
	for i, m := range matches {
		result[i] = map[string]string{
			"name": m.name,
			"path": m.path,
			"type": m.typ,
		}
	}

	s.writeJSON(w, r, result)
}

func (s *webServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			ContextMessagePercent      float64 `json:"context_message_percent"`
			ContextConversationPercent float64 `json:"context_conversation_percent"`
			ContextProtectRecent       int     `json:"context_protect_recent"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		// Validate inputs
		if req.ContextMessagePercent <= 0 || req.ContextMessagePercent > 0.10 {
			s.respondError(w, r, http.StatusBadRequest, "context_message_percent must be between 0 and 0.10 (0-10%)")
			return
		}
		if req.ContextConversationPercent <= 0 || req.ContextConversationPercent > 0.80 {
			s.respondError(w, r, http.StatusBadRequest, "context_conversation_percent must be between 0 and 0.80 (0-80%)")
			return
		}
		if req.ContextProtectRecent < 0 {
			s.respondError(w, r, http.StatusBadRequest, "context_protect_recent must be >= 0")
			return
		}

		// Update config
		s.agent.cfg.ContextMessagePercent = req.ContextMessagePercent
		s.agent.cfg.ContextTotalPercent = req.ContextConversationPercent
		s.agent.cfg.ContextProtectRecent = req.ContextProtectRecent

		// Save to config file
		if err := config.Save(s.agent.cfg); err != nil {
			s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to save config: %v", err))
			return
		}

		// Reload all workspace profiles with new config
		s.agent.workspacesMu.RLock()
		for _, wsCtx := range s.agent.workspaceContexts {
			if reloadable, ok := wsCtx.profile.(interface {
				ReloadConfig(config.Config) error
			}); ok {
				if err := reloadable.ReloadConfig(s.agent.cfg); err != nil {
					s.agent.workspacesMu.RUnlock()
					s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to reload workspace profile: %v", err))
					return
				}
			}
		}
		s.agent.workspacesMu.RUnlock()

		// Also reload default profile for CLI mode
		if reloadable, ok := s.agent.profile.(interface {
			ReloadConfig(config.Config) error
		}); ok {
			if err := reloadable.ReloadConfig(s.agent.cfg); err != nil {
				s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to reload profile: %v", err))
				return
			}
		}

		s.writeJSON(w, r, map[string]string{"status": "saved"})
		return
	}

	s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
}

func (s *webServer) handleSystemPrompt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid payload")
		return
	}
	trimmed := strings.TrimSpace(req.Prompt)
	s.agent.UpdateSystemPrompt(trimmed)

	if err := config.Save(s.agent.cfg); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to save system prompt: %v", err))
		return
	}
	s.writeJSON(w, r, map[string]any{
		"system_prompt": trimmed,
	})
}

// handleWorkspaces returns the list of all workspaces
func (s *webServer) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.workspaceManager == nil {
		s.respondError(w, r, http.StatusInternalServerError, "workspace manager not initialized")
		return
	}

	workspaces := s.workspaceManager.List()
	current := s.workspaceManager.Current()

	s.writeJSON(w, r, map[string]interface{}{
		"workspaces": workspaces,
		"current":    current,
	})
}

// handleWorkspaceAdd adds a new workspace
func (s *webServer) handleWorkspaceAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.workspaceManager == nil {
		s.respondError(w, r, http.StatusInternalServerError, "workspace manager not initialized")
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if req.Path == "" {
		s.respondError(w, r, http.StatusBadRequest, "path is required")
		return
	}

	workspace, err := s.workspaceManager.Add(req.Path)
	if err != nil {
		s.respondError(w, r, http.StatusBadRequest, fmt.Sprintf("failed to add workspace: %v", err))
		return
	}

	s.writeJSON(w, r, map[string]interface{}{
		"workspace": workspace,
		"status":    "added",
	})
}

// handleWorkspaceSwitch switches to a different workspace
func (s *webServer) handleWorkspaceSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.workspaceManager == nil {
		s.respondError(w, r, http.StatusInternalServerError, "workspace manager not initialized")
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if req.Path == "" {
		s.respondError(w, r, http.StatusBadRequest, "path is required")
		return
	}

	// Switch workspace in manager (updates current)
	if err := s.workspaceManager.SetCurrent(req.Path); err != nil {
		s.respondError(w, r, http.StatusBadRequest, fmt.Sprintf("failed to set current workspace: %v", err))
		return
	}

	// NOTE: This endpoint is deprecated - frontend now manages workspace via localStorage
	// But keeping for backward compatibility

	// Return new session data for requested workspace
	payload, err := s.buildSessionPayload(r.Context(), req.Path)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to build session: %v", err))
		return
	}
	s.writeJSON(w, r, map[string]interface{}{
		"status":  "switched",
		"session": payload,
	})
}

// handleWorkspaceRemove removes a workspace
func (s *webServer) handleWorkspaceRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.workspaceManager == nil {
		s.respondError(w, r, http.StatusInternalServerError, "workspace manager not initialized")
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if req.Path == "" {
		s.respondError(w, r, http.StatusBadRequest, "path is required")
		return
	}

	// Remove from workspace list
	if err := s.workspaceManager.Remove(req.Path); err != nil {
		s.respondError(w, r, http.StatusBadRequest, fmt.Sprintf("failed to remove workspace: %v", err))
		return
	}

	// If removed workspace was current, switch to new current
	current := s.workspaceManager.Current()
	if current != nil && current.Path != s.agent.workspaceRoot {
		if err := s.agent.SwitchWorkspace(current.Path); err != nil {
			s.logger.Printf("Warning: failed to switch to new current workspace: %v", err)
		}
	}

	s.writeJSON(w, r, map[string]interface{}{
		"status":  "removed",
		"current": current,
	})
}

// handleBrowse returns a list of directories at the requested path
func (s *webServer) handleBrowse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get path from query param
	reqPath := r.URL.Query().Get("path")

	// Default to current workspace or home directory
	if reqPath == "" {
		// Get current workspace from request context (X-Workspace header or workspaceManager)
		workspacePath := s.getWorkspaceFromRequest(r)
		if workspacePath != "" {
			reqPath = workspacePath
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				s.respondError(w, r, http.StatusInternalServerError, "failed to get home directory")
				return
			}
			reqPath = homeDir
		}
	}

	// Expand ~ to home directory
	if strings.HasPrefix(reqPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			s.respondError(w, r, http.StatusBadRequest, "failed to expand home directory")
			return
		}
		reqPath = filepath.Join(homeDir, reqPath[1:])
	}

	// Clean and validate path
	absPath, err := filepath.Abs(reqPath)
	if err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid path")
		return
	}

	// Check if path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		s.logger.Printf("Browse error - stat failed for %s: %v", absPath, err)
		s.respondError(w, r, http.StatusNotFound, fmt.Sprintf("path does not exist: %v", err))
		return
	}
	if !info.IsDir() {
		s.logger.Printf("Browse error - not a directory: %s", absPath)
		s.respondError(w, r, http.StatusBadRequest, "path is not a directory")
		return
	}

	// Read directory entries
	entries, err := os.ReadDir(absPath)
	if err != nil {
		s.logger.Printf("Browse error - readdir failed for %s: %v", absPath, err)
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to read directory: %v", err))
		return
	}

	// Filter to only directories, exclude hidden
	type DirEntry struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}

	// Initialize as empty slice (not nil) so JSON marshals to [] instead of null
	dirs := make([]DirEntry, 0)
	for _, entry := range entries {
		// Skip non-directories
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories (starting with .)
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		dirs = append(dirs, DirEntry{
			Name: entry.Name(),
			Path: filepath.Join(absPath, entry.Name()),
		})
	}

	// Sort alphabetically
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Name < dirs[j].Name
	})

	// Get parent directory
	parentPath := filepath.Dir(absPath)

	s.writeJSON(w, r, map[string]interface{}{
		"current":     absPath,
		"parent":      parentPath,
		"directories": dirs,
	})
}

// handleFolderCreate creates a new directory at the specified path
func (s *webServer) handleFolderCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if req.Path == "" {
		s.respondError(w, r, http.StatusBadRequest, "path is required")
		return
	}

	// Clean and validate path
	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		s.respondError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid path: %v", err))
		return
	}

	// Security check: prevent creating folders outside user home or absolute paths with ..
	homeDir, err := os.UserHomeDir()
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, "failed to get home directory")
		return
	}
	if !strings.HasPrefix(absPath, homeDir) && !strings.HasPrefix(absPath, "/tmp") {
		s.respondError(w, r, http.StatusForbidden, "cannot create folders outside home directory")
		return
	}

	// Check if path already exists
	if _, err := os.Stat(absPath); err == nil {
		s.respondError(w, r, http.StatusConflict, "path already exists")
		return
	}

	// Create directory with standard permissions
	if err := os.MkdirAll(absPath, 0755); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to create directory: %v", err))
		return
	}

	s.logger.Printf("Created folder: %s", absPath)

	s.writeJSON(w, r, map[string]interface{}{
		"path":   absPath,
		"status": "created",
	})
}

// findAvailableBranchName generates a simple branch name suffix (-1, -2, -a, -b, etc)
func findAvailableBranchName(states *state.Manager, baseKey string) string {
	existingKeys := states.ListKeys()
	keyMap := make(map[string]bool)
	for _, k := range existingKeys {
		keyMap[k] = true
	}

	// Try numeric suffixes first: -1, -2, -3, ...
	for i := 1; i <= 99; i++ {
		candidate := fmt.Sprintf("%s-%d", baseKey, i)
		if !keyMap[candidate] {
			return candidate
		}
	}

	// Fallback to letter suffixes: -a, -b, -c, ...
	for c := 'a'; c <= 'z'; c++ {
		candidate := fmt.Sprintf("%s-%c", baseKey, c)
		if !keyMap[candidate] {
			return candidate
		}
	}

	// Ultimate fallback with timestamp
	return fmt.Sprintf("%s-branch", baseKey)
}

// handleBranch creates a new session by branching from current session at a specific message index
func (s *webServer) handleBranch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		EditIndex  int    `json:"edit_index"`
		NewContent string `json:"new_content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if req.NewContent == "" {
		s.respondError(w, r, http.StatusBadRequest, "new_content is required")
		return
	}

	// Get workspace context
	workspace := s.getWorkspaceFromRequest(r)
	if workspace == "" || !s.workspaceExists(workspace) {
		s.respondError(w, r, http.StatusBadRequest, "select a workspace first")
		return
	}

	wsCtx, err := s.agent.GetOrCreateWorkspaceContext(workspace)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("get workspace context: %v", err))
		return
	}

	// Get current conversation
	currentConv := wsCtx.states.Current()
	currentMessages := currentConv.Messages()

	if req.EditIndex < 0 || req.EditIndex >= len(currentMessages) {
		s.respondError(w, r, http.StatusBadRequest, "invalid edit_index")
		return
	}

	// Generate simple session name with counter suffix
	currentKey := currentConv.Key()
	newKey := findAvailableBranchName(wsCtx.states, currentKey)

	// Create new session
	newConv, err := wsCtx.states.NewState(newKey)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to create new session: %v", err))
		return
	}

	// Copy messages up to (not including) the edit index
	// Do NOT include the edited message - frontend will submit it
	messagesToCopy := currentMessages[0:req.EditIndex]

	// Set messages in new conversation
	newConv.ReplaceMessages(messagesToCopy)

	// Save new conversation
	if err := wsCtx.states.Save(newConv); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to save new session: %v", err))
		return
	}

	s.writeJSON(w, r, map[string]interface{}{
		"new_session_key": newKey,
		"status":          "branched",
	})
}
