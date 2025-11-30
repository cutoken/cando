package agent

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cando/internal/analytics"
	"cando/internal/config"
	"cando/internal/contextprofile"
	"cando/internal/credentials"
	"cando/internal/llm"
	"cando/internal/logging"
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

// OpenRouter models cache for daily refresh
const (
	openrouterModelsURL     = "https://openrouter.ai/api/frontend/models/find?order=top-weekly"
	openrouterCacheDuration = 24 * time.Hour
	openrouterFetchTimeout  = 30 * time.Second
)

type openrouterModelCache struct {
	data      []byte
	fetchedAt time.Time
	mu        sync.RWMutex
}

var orModelCache = &openrouterModelCache{}

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
	httpServer       *http.Server
	shutdownCh       chan struct{}
	binaryPath       string // Original binary path, captured at startup for restart
}

func (s *webServer) run(ctx context.Context) error {
	if s.logger == nil {
		s.logger = log.Default()
	}

	// Capture original binary path at startup (before any renames during update)
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			s.binaryPath = resolved
		} else {
			s.binaryPath = exe
		}
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
	mux.HandleFunc("/api/project/instructions", s.handleProjectInstructions)
	mux.HandleFunc("/api/plan-mode", s.handlePlanMode)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/update-check", s.handleUpdateCheck)
	mux.HandleFunc("/api/update", s.handleUpdate)
	mux.HandleFunc("/api/restart", s.handleRestart)
	mux.HandleFunc("/api/update/dismiss", s.handleUpdateDismiss)
	mux.HandleFunc("/api/telemetry", s.handleTelemetry)
	mux.HandleFunc("/api/files/tree", s.handleFilesTree)
	mux.HandleFunc("/api/files/read", s.handleFilesRead)
	mux.HandleFunc("/api/files/save", s.handleFilesSave)
	mux.HandleFunc("/api/files/create", s.handleFilesCreate)
	mux.HandleFunc("/api/files/mkdir", s.handleFilesMkdir)
	mux.HandleFunc("/api/files/reveal", s.handleFilesReveal)

	server := &http.Server{
		Addr:    actualAddr,
		Handler: s.logRequests(mux),
	}
	s.httpServer = server
	s.shutdownCh = make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
		case <-s.shutdownCh:
		}
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
	// High-frequency polling endpoints to skip logging
	skipLogPaths := map[string]bool{
		"/api/files/tree": true,
		"/api/health":     true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		workspace := s.getWorkspaceFromRequest(r)
		next.ServeHTTP(w, r)

		// Skip logging for polling endpoints to reduce noise
		if skipLogPaths[r.URL.Path] {
			return
		}

		duration := time.Since(start).Round(time.Millisecond)
		if workspace != "" {
			s.logger.Printf("[ws:%s] [%s] %s %s (%s)", workspace, r.RemoteAddr, r.Method, r.URL.Path, duration)
		} else {
			s.logger.Printf("[%s] %s %s (%s)", r.RemoteAddr, r.Method, r.URL.Path, duration)
		}
	})
}

func (s *webServer) logRequestError(r *http.Request, status int, message string) {
	if s.logger == nil {
		return
	}
	workspace := s.getWorkspaceFromRequest(r)
	if workspace != "" {
		s.logger.Printf("[ERROR] [ws:%s] status=%d method=%s path=%s remote=%s: %s",
			workspace, status, r.Method, r.URL.Path, r.RemoteAddr, message)
	} else {
		s.logger.Printf("[ERROR] status=%d method=%s path=%s remote=%s: %s",
			status, r.Method, r.URL.Path, r.RemoteAddr, message)
	}
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
	data := s.getOpenRouterModels()
	_, _ = w.Write(data)
}

// getOpenRouterModels returns cached models, fetches fresh if stale, falls back to embedded
func (s *webServer) getOpenRouterModels() []byte {
	orModelCache.mu.RLock()
	if time.Since(orModelCache.fetchedAt) < openrouterCacheDuration && len(orModelCache.data) > 0 {
		defer orModelCache.mu.RUnlock()
		return orModelCache.data
	}
	orModelCache.mu.RUnlock()

	// Try to fetch fresh data in background if cache exists but stale
	if len(orModelCache.data) > 0 {
		go s.refreshOpenRouterModels()
		orModelCache.mu.RLock()
		defer orModelCache.mu.RUnlock()
		return orModelCache.data
	}

	// No cache - fetch synchronously
	data, err := fetchOpenRouterModels()
	if err != nil {
		logging.DevLog("openrouter models fetch failed: %v, using embedded fallback", err)
		return openrouterModels
	}

	orModelCache.mu.Lock()
	orModelCache.data = data
	orModelCache.fetchedAt = time.Now()
	orModelCache.mu.Unlock()

	logging.DevLog("openrouter models fetched: %d bytes", len(data))
	return data
}

func (s *webServer) refreshOpenRouterModels() {
	data, err := fetchOpenRouterModels()
	if err != nil {
		logging.DevLog("openrouter models background refresh failed: %v", err)
		return
	}

	orModelCache.mu.Lock()
	orModelCache.data = data
	orModelCache.fetchedAt = time.Now()
	orModelCache.mu.Unlock()
	logging.DevLog("openrouter models refreshed: %d bytes", len(data))
}

// fetchOpenRouterModels fetches and transforms models from OpenRouter API
func fetchOpenRouterModels() ([]byte, error) {
	client := &http.Client{Timeout: openrouterFetchTimeout}
	resp, err := client.Get(openrouterModelsURL)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	// Parse the response and transform to our format
	var apiResp struct {
		Data struct {
			Models []struct {
				Name            string   `json:"name"`
				InputModalities []string `json:"input_modalities"`
				HasTextOutput   bool     `json:"has_text_output"`
				Endpoint        struct {
					ModelVariantSlug string `json:"model_variant_slug"`
					Pricing          struct {
						Prompt     string `json:"prompt"`
						Completion string `json:"completion"`
					} `json:"pricing"`
				} `json:"endpoint"`
			} `json:"models"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse failed: %w", err)
	}

	// Transform to our format
	type modelEntry struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Capabilities []string `json:"capabilities"`
		Pricing      struct {
			Prompt     string `json:"prompt"`
			Completion string `json:"completion"`
		} `json:"pricing"`
	}

	var models []modelEntry
	for _, m := range apiResp.Data.Models {
		if !m.HasTextOutput || m.Endpoint.ModelVariantSlug == "" {
			continue
		}
		models = append(models, modelEntry{
			ID:           m.Endpoint.ModelVariantSlug,
			Name:         m.Name,
			Capabilities: m.InputModalities,
			Pricing: struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			}{
				Prompt:     m.Endpoint.Pricing.Prompt,
				Completion: m.Endpoint.Pricing.Completion,
			},
		})
	}

	// Sanity check: if we got very few models, API structure likely changed
	if len(models) < 100 {
		return nil, fmt.Errorf("too few models returned (%d), API structure may have changed", len(models))
	}

	return json.Marshal(models)
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
	s.agent.logger.Printf("[ws:%s] handleStream: starting conversation", workspace)
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
		// Check if this is a structured ProviderError (event may already have been sent by agent)
		if pe, ok := llm.IsProviderError(err); ok {
			// Log with provider context instead of generic ERROR
			s.logger.Printf("[provider] %s error: %s", pe.Provider, pe.Error())
			// Send provider_error event (in case it wasn't sent during retry loop)
			sendEvent("provider_error", buildProviderErrorPayload(pe))
		} else {
			// Generic error handling
			s.logRequestError(r, http.StatusInternalServerError, fmt.Sprintf("stream request failed: %v", err))
			sendEvent("error", map[string]string{"message": err.Error()})
		}
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
	s.agent.logger.Printf("[ws:%s] handleState: action=%s, key=%s", workspace, req.Action, req.Key)
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
	PlanMode              bool              `json:"plan_mode"`
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
	OpenRouterFreeMode    bool              `json:"openrouter_free_mode,omitempty"`
	AnalyticsEnabled      bool              `json:"analytics_enabled"`
	ContextProfile        string            `json:"context_profile,omitempty"`
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
		s.logger.Printf("[ws:%s] workspace not found; returning empty state", workspace)
		workspace = ""
	}
	if workspace != "" {
		s.agent.logger.Printf("[ws:%s] writeSessionPayload: building session", workspace)
	} else {
		s.agent.logger.Printf("writeSessionPayload: no workspace selected")
	}
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
		Thinking:              s.agent.cfg.ThinkingEnabled, // Use config value, not agent cache
		ForceThinking:         s.agent.cfg.ForceThinking,   // Use config value, not agent cache
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
		OpenRouterFreeMode:    s.agent.cfg.OpenRouterFreeMode,
		AnalyticsEnabled:      s.agent.cfg.IsAnalyticsEnabled(),
		ContextProfile:        s.agent.cfg.ContextProfile, // Add missing field for profile dropdown
		Config: &configSnapshot{ // Global config should always be available
			ContextProfile:             s.agent.cfg.ContextProfile,
			ContextMessagePercent:      s.agent.cfg.ContextMessagePercent,
			ContextConversationPercent: s.agent.cfg.ContextTotalPercent,
			ContextProtectRecent:       s.agent.cfg.ContextProtectRecent,
			SystemPrompt:               s.agent.cfg.SystemPrompt,
		},
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
	payload.PlanMode = wsCtx.planMode
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
			ContextMessagePercent      *float64 `json:"context_message_percent"`
			ContextConversationPercent *float64 `json:"context_conversation_percent"`
			ContextProtectRecent       *int     `json:"context_protect_recent"`
			OpenRouterFreeMode         *bool    `json:"openrouter_free_mode"`
			AnalyticsEnabled           *bool    `json:"analytics_enabled"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondError(w, r, http.StatusBadRequest, err.Error())
			return
		}

		// Validate and update compaction settings if provided
		if req.ContextMessagePercent != nil {
			if *req.ContextMessagePercent <= 0 || *req.ContextMessagePercent > 0.10 {
				s.respondError(w, r, http.StatusBadRequest, "context_message_percent must be between 0 and 0.10 (0-10%)")
				return
			}
			s.agent.cfg.ContextMessagePercent = *req.ContextMessagePercent
		}
		if req.ContextConversationPercent != nil {
			if *req.ContextConversationPercent <= 0 || *req.ContextConversationPercent > 0.80 {
				s.respondError(w, r, http.StatusBadRequest, "context_conversation_percent must be between 0 and 0.80 (0-80%)")
				return
			}
			s.agent.cfg.ContextTotalPercent = *req.ContextConversationPercent
		}
		if req.ContextProtectRecent != nil {
			if *req.ContextProtectRecent < 0 {
				s.respondError(w, r, http.StatusBadRequest, "context_protect_recent must be >= 0")
				return
			}
			s.agent.cfg.ContextProtectRecent = *req.ContextProtectRecent
		}

		// Update OpenRouter Free Mode if provided
		if req.OpenRouterFreeMode != nil {
			s.agent.cfg.OpenRouterFreeMode = *req.OpenRouterFreeMode
		}

		// Update Analytics if provided
		if req.AnalyticsEnabled != nil {
			s.agent.cfg.AnalyticsEnabled = req.AnalyticsEnabled
			analytics.SetEnabled(*req.AnalyticsEnabled)
		}

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

	// Check if we should include files
	includeFiles := r.URL.Query().Get("includeFiles") == "true"

	// Filter entries
	type DirEntry struct {
		Name  string `json:"name"`
		Path  string `json:"path"`
		IsDir bool   `json:"isDir"`
	}

	// Initialize as empty slice (not nil) so JSON marshals to [] instead of null
	dirs := make([]DirEntry, 0)
	files := make([]DirEntry, 0)

	for _, entry := range entries {
		// Skip hidden entries (starting with .)
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		fullPath := filepath.Join(absPath, entry.Name())

		if entry.IsDir() {
			dirs = append(dirs, DirEntry{
				Name:  entry.Name(),
				Path:  fullPath,
				IsDir: true,
			})
		} else if includeFiles {
			files = append(files, DirEntry{
				Name:  entry.Name(),
				Path:  fullPath,
				IsDir: false,
			})
		}
	}

	// Sort alphabetically
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Name < dirs[j].Name
	})
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	// Get parent directory
	parentPath := filepath.Dir(absPath)

	response := map[string]interface{}{
		"current":     absPath,
		"parent":      parentPath,
		"directories": dirs,
	}
	if includeFiles {
		response["files"] = files
	}

	s.writeJSON(w, r, response)
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

// handleProjectInstructions handles GET/POST for project-level instructions
func (s *webServer) handleProjectInstructions(w http.ResponseWriter, r *http.Request) {
	workspacePath := r.Header.Get("X-Workspace")
	if workspacePath == "" {
		s.respondError(w, r, http.StatusBadRequest, "workspace header required")
		return
	}

	// Get project storage root
	storageRoot, err := ProjectStorageRoot(workspacePath)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to get storage root: %v", err))
		return
	}

	instructionsPath := filepath.Join(storageRoot, "instructions.txt")

	switch r.Method {
	case http.MethodGet:
		// Read instructions file
		content, err := os.ReadFile(instructionsPath)
		if err != nil {
			if os.IsNotExist(err) {
				// No instructions file yet, return empty
				s.writeJSON(w, r, map[string]string{"instructions": ""})
				return
			}
			s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to read instructions: %v", err))
			return
		}
		s.writeJSON(w, r, map[string]string{"instructions": string(content)})

	case http.MethodPost:
		var req struct {
			Instructions string `json:"instructions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondError(w, r, http.StatusBadRequest, "invalid request body")
			return
		}

		// Ensure storage directory exists
		if err := os.MkdirAll(storageRoot, 0o755); err != nil {
			s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to create storage dir: %v", err))
			return
		}

		// Write instructions file
		if err := os.WriteFile(instructionsPath, []byte(req.Instructions), 0o644); err != nil {
			s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to write instructions: %v", err))
			return
		}

		s.writeJSON(w, r, map[string]string{"status": "saved"})

	default:
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *webServer) handlePlanMode(w http.ResponseWriter, r *http.Request) {
	workspacePath := r.Header.Get("X-Workspace")
	if workspacePath == "" {
		s.respondError(w, r, http.StatusBadRequest, "workspace header required")
		return
	}

	// Get or create workspace context
	wsCtx, err := s.agent.GetOrCreateWorkspaceContext(workspacePath)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to get workspace: %v", err))
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, r, map[string]bool{"planMode": wsCtx.planMode})

	case http.MethodPost:
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.respondError(w, r, http.StatusBadRequest, "invalid request body")
			return
		}

		wsCtx.planMode = req.Enabled
		s.writeJSON(w, r, map[string]bool{"planMode": wsCtx.planMode})

	default:
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// Update-related constants
const (
	githubRepoOwner       = "cutoken"
	githubRepoName        = "cando"
	updateCheckTimeout    = 10 * time.Second
	updateDownloadTimeout = 120 * time.Second
)

func (s *webServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, r, map[string]string{"status": "ok"})
}

func (s *webServer) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	currentVersion := s.agent.version
	forceCheck := r.URL.Query().Get("force") == "true"
	isDev := currentVersion == "" || currentVersion == "dev"

	// Load update state
	state, err := config.LoadUpdateState()
	if err != nil {
		s.logger.Printf("failed to load update state: %v", err)
		state = &config.UpdateState{}
	}

	// Check if dismissed (skip if force check or dev)
	if !isDev && !forceCheck && state.IsDismissed() {
		s.writeJSON(w, r, map[string]any{
			"current":         currentVersion,
			"latest":          state.LatestVersion,
			"updateAvailable": false,
			"dismissed":       true,
			"isDev":           false,
		})
		return
	}

	// Check if we need to fetch new version info
	latestVersion := state.LatestVersion
	if forceCheck || state.NeedsCheck() {
		if fetched, err := s.fetchLatestVersion(); err != nil {
			s.logger.Printf("failed to fetch latest version: %v", err)
		} else {
			latestVersion = fetched
			state.RecordCheck(latestVersion)
			if err := state.Save(); err != nil {
				s.logger.Printf("failed to save update state: %v", err)
			}
		}
	}

	// Dev versions can see latest but can't update
	updateAvailable := !isDev && latestVersion != "" && compareVersions(latestVersion, currentVersion) > 0

	s.writeJSON(w, r, map[string]any{
		"current":         currentVersion,
		"latest":          latestVersion,
		"updateAvailable": updateAvailable,
		"dismissed":       false,
		"isDev":           isDev,
	})
}

func (s *webServer) handleUpdateDismiss(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	state, err := config.LoadUpdateState()
	if err != nil {
		state = &config.UpdateState{}
	}

	state.Dismiss()
	if err := state.Save(); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to save state: %v", err))
		return
	}

	s.writeJSON(w, r, map[string]string{"status": "dismissed"})
}

func (s *webServer) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserAgent  string `json:"user_agent"`
		ScreenSize string `json:"screen_size"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid request")
		return
	}

	// Update browser context for analytics
	analytics.SetBrowserContext(req.UserAgent, req.ScreenSize)

	// Track page view now that we have browser context
	analytics.TrackPageView()

	s.writeJSON(w, r, map[string]string{"status": "ok"})
}

// FileTreeEntry represents a file or directory in the tree
type FileTreeEntry struct {
	Name     string          `json:"name"`
	Path     string          `json:"path"`
	IsDir    bool            `json:"isDir"`
	Children []FileTreeEntry `json:"children,omitempty"`
}

func (s *webServer) handleFilesTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	workspacePath := r.URL.Query().Get("workspace")
	if workspacePath == "" {
		s.respondError(w, r, http.StatusBadRequest, "workspace parameter required")
		return
	}

	// Validate workspace path exists
	info, err := os.Stat(workspacePath)
	if err != nil || !info.IsDir() {
		s.respondError(w, r, http.StatusBadRequest, "invalid workspace path")
		return
	}

	// Build the file tree (limited depth to avoid huge responses)
	tree, err := s.buildFileTree(workspacePath, workspacePath, 0, 5)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to read directory: %v", err))
		return
	}

	s.writeJSON(w, r, tree)
}

func (s *webServer) buildFileTree(basePath, currentPath string, depth, maxDepth int) ([]FileTreeEntry, error) {
	if depth >= maxDepth {
		return []FileTreeEntry{}, nil
	}

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return nil, err
	}

	result := []FileTreeEntry{}
	for _, entry := range entries {
		name := entry.Name()

		// Skip large/generated directories only
		if entry.IsDir() {
			switch name {
			case "node_modules", "vendor", "__pycache__", ".git", ".svn", ".hg", ".next", ".nuxt", "target", "bin", "obj":
				continue
			}
		}

		fullPath := filepath.Join(currentPath, name)
		relPath, _ := filepath.Rel(basePath, fullPath)

		item := FileTreeEntry{
			Name:  name,
			Path:  relPath,
			IsDir: entry.IsDir(),
		}

		if entry.IsDir() {
			children, err := s.buildFileTree(basePath, fullPath, depth+1, maxDepth)
			if err == nil {
				item.Children = children
			}
		}

		result = append(result, item)
	}

	// Sort: directories first, then alphabetically
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result, nil
}

func (s *webServer) handleFilesRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	workspacePath := r.URL.Query().Get("workspace")
	filePath := r.URL.Query().Get("path")

	if workspacePath == "" || filePath == "" {
		s.respondError(w, r, http.StatusBadRequest, "workspace and path parameters required")
		return
	}

	// Construct full path and validate it's within workspace
	fullPath := filepath.Join(workspacePath, filePath)
	fullPath = filepath.Clean(fullPath)

	if !strings.HasPrefix(fullPath, filepath.Clean(workspacePath)) {
		s.respondError(w, r, http.StatusForbidden, "path traversal not allowed")
		return
	}

	// Check file exists and is not a directory
	info, err := os.Stat(fullPath)
	if err != nil {
		s.respondError(w, r, http.StatusNotFound, "file not found")
		return
	}
	if info.IsDir() {
		s.respondError(w, r, http.StatusBadRequest, "cannot read directory")
		return
	}

	// Check file size (limit to 1MB)
	if info.Size() > 1024*1024 {
		s.respondError(w, r, http.StatusBadRequest, "file too large (max 1MB)")
		return
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to read file: %v", err))
		return
	}

	// Detect if binary
	isBinary := false
	for _, b := range content[:min(512, len(content))] {
		if b == 0 {
			isBinary = true
			break
		}
	}

	s.writeJSON(w, r, map[string]interface{}{
		"path":     filePath,
		"content":  string(content),
		"isBinary": isBinary,
		"size":     info.Size(),
		"modTime":  info.ModTime().Unix(),
	})
}

func (s *webServer) handleFilesSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Workspace string `json:"workspace"`
		Path      string `json:"path"`
		Content   string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Workspace == "" || req.Path == "" {
		s.respondError(w, r, http.StatusBadRequest, "workspace and path required")
		return
	}

	// Construct full path and validate it's within workspace
	fullPath := filepath.Join(req.Workspace, req.Path)
	fullPath = filepath.Clean(fullPath)

	if !strings.HasPrefix(fullPath, filepath.Clean(req.Workspace)) {
		s.respondError(w, r, http.StatusForbidden, "path traversal not allowed")
		return
	}

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to create directory: %v", err))
		return
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to write file: %v", err))
		return
	}

	// Get updated file info
	info, _ := os.Stat(fullPath)
	modTime := int64(0)
	if info != nil {
		modTime = info.ModTime().Unix()
	}

	s.writeJSON(w, r, map[string]interface{}{
		"status":  "saved",
		"path":    req.Path,
		"modTime": modTime,
	})
}

func (s *webServer) handleFilesCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Workspace string `json:"workspace"`
		Path      string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Workspace == "" || req.Path == "" {
		s.respondError(w, r, http.StatusBadRequest, "workspace and path required")
		return
	}

	fullPath := filepath.Join(req.Workspace, req.Path)
	fullPath = filepath.Clean(fullPath)

	if !strings.HasPrefix(fullPath, filepath.Clean(req.Workspace)) {
		s.respondError(w, r, http.StatusForbidden, "path traversal not allowed")
		return
	}

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		s.respondError(w, r, http.StatusConflict, "file already exists")
		return
	}

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to create directory: %v", err))
		return
	}

	// Create empty file
	if err := os.WriteFile(fullPath, []byte{}, 0644); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to create file: %v", err))
		return
	}

	s.writeJSON(w, r, map[string]interface{}{
		"status": "created",
		"path":   req.Path,
	})
}

func (s *webServer) handleFilesMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Workspace string `json:"workspace"`
		Path      string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Workspace == "" || req.Path == "" {
		s.respondError(w, r, http.StatusBadRequest, "workspace and path required")
		return
	}

	fullPath := filepath.Join(req.Workspace, req.Path)
	fullPath = filepath.Clean(fullPath)

	if !strings.HasPrefix(fullPath, filepath.Clean(req.Workspace)) {
		s.respondError(w, r, http.StatusForbidden, "path traversal not allowed")
		return
	}

	// Check if directory already exists
	if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
		s.respondError(w, r, http.StatusConflict, "directory already exists")
		return
	}

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to create directory: %v", err))
		return
	}

	s.writeJSON(w, r, map[string]interface{}{
		"status": "created",
		"path":   req.Path,
	})
}

func (s *webServer) handleFilesReveal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Workspace string `json:"workspace"`
		Path      string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Workspace == "" {
		s.respondError(w, r, http.StatusBadRequest, "workspace required")
		return
	}

	// If path is empty, reveal the workspace folder itself
	fullPath := req.Workspace
	if req.Path != "" {
		fullPath = filepath.Join(req.Workspace, req.Path)
	}
	fullPath = filepath.Clean(fullPath)

	if !strings.HasPrefix(fullPath, filepath.Clean(req.Workspace)) {
		s.respondError(w, r, http.StatusForbidden, "path traversal not allowed")
		return
	}

	// Check if path exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		s.respondError(w, r, http.StatusNotFound, "path not found")
		return
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", fullPath)
	case "windows":
		cmd = exec.Command("explorer", "/select,", fullPath)
	default: // linux and others
		// xdg-open opens the parent directory for files
		cmd = exec.Command("xdg-open", filepath.Dir(fullPath))
	}

	if err := cmd.Start(); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to open explorer: %v", err))
		return
	}

	s.writeJSON(w, r, map[string]interface{}{
		"status": "opened",
		"path":   req.Path,
	})
}

func (s *webServer) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	currentVersion := s.agent.version
	if currentVersion == "" || currentVersion == "dev" {
		s.respondError(w, r, http.StatusBadRequest, "cannot update dev version")
		return
	}

	// Get current binary path
	binaryPath, err := os.Executable()
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to get executable path: %v", err))
		return
	}
	binaryPath, err = filepath.EvalSymlinks(binaryPath)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to resolve symlinks: %v", err))
		return
	}

	// Check if directory is writable (we rename files, not write to running binary)
	if err := checkDirWritable(filepath.Dir(binaryPath)); err != nil {
		s.writeJSON(w, r, map[string]any{
			"success":     false,
			"needsManual": true,
			"command":     getManualInstallCommand(),
			"message":     fmt.Sprintf("Directory not writable: %v", err),
		})
		return
	}

	// Download new binary
	s.logger.Printf("downloading update...")
	if err := s.downloadAndReplaceBinary(binaryPath); err != nil {
		s.respondError(w, r, http.StatusInternalServerError, fmt.Sprintf("update failed: %v", err))
		return
	}

	// Clear update state so new version doesn't show update prompt
	state, _ := config.LoadUpdateState()
	state.LatestVersion = "" // Clear so new binary checks fresh
	state.DismissedUntil = time.Time{}
	state.LastCheckTime = time.Time{}
	_ = state.Save()

	s.logger.Printf("update downloaded successfully")
	s.writeJSON(w, r, map[string]any{
		"success": true,
		"message": "Update downloaded. Click Restart to apply.",
	})
}

func (s *webServer) handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	s.triggerRestart(w, r)
}

func (s *webServer) triggerRestart(w http.ResponseWriter, r *http.Request) {
	// Send response before restarting
	s.writeJSON(w, r, map[string]string{"status": "restarting"})

	// Give time for response to be sent, then exec directly
	// We do exec BEFORE shutdown to avoid race with main() exiting
	go func() {
		time.Sleep(500 * time.Millisecond)

		// Print to stdout so user sees it in terminal
		fmt.Println("\n🔄 Restarting Cando...")
		s.logger.Printf("restarting application...")

		// Use the original binary path captured at startup
		// (os.Executable() would return the .backup path after rename during update)
		binary := s.binaryPath
		if binary == "" {
			// Fallback if not set (shouldn't happen)
			var err error
			binary, err = os.Executable()
			if err != nil {
				fmt.Printf("❌ Failed to get executable: %v\n", err)
				s.logger.Printf("failed to get executable: %v", err)
				os.Exit(1)
			}
		}

		s.logger.Printf("exec: %s %v", binary, os.Args)

		// Add env var to signal this is a restart (skip browser open)
		env := os.Environ()
		env = append(env, "CANDO_RESTARTING=1")

		// syscall.Exec replaces current process, no need for graceful shutdown
		// The OS will clean up the listening socket
		if err := execBinary(binary, os.Args, env); err != nil {
			fmt.Printf("❌ Failed to restart: %v\n", err)
			s.logger.Printf("failed to exec: %v", err)
			os.Exit(1)
		}
	}()
}

func (s *webServer) fetchLatestVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()

	// Check if running as beta version (include prereleases in update check)
	binaryName := filepath.Base(s.binaryPath)
	if binaryName == "" || binaryName == "." {
		// Fallback: try to get executable path directly
		if exe, err := os.Executable(); err == nil {
			binaryName = filepath.Base(exe)
		}
	}
	isBeta := strings.Contains(strings.ToLower(binaryName), "beta")
	s.logger.Printf("update check: binary=%s, isBeta=%v", binaryName, isBeta)

	var url string
	if isBeta {
		// Beta channel: fetch releases and find latest prerelease
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=10", githubRepoOwner, githubRepoName)
	} else {
		// Stable channel: fetch only latest stable release
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubRepoOwner, githubRepoName)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	if isBeta {
		// Parse array response for /releases endpoint, find latest prerelease
		var releases []struct {
			TagName    string `json:"tag_name"`
			Prerelease bool   `json:"prerelease"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return "", err
		}
		s.logger.Printf("beta channel: fetched %d releases", len(releases))
		// Find the first prerelease
		for _, r := range releases {
			if r.Prerelease {
				s.logger.Printf("beta channel: found prerelease %s", r.TagName)
				return r.TagName, nil
			}
		}
		// No prerelease found - this shouldn't happen if releases exist
		s.logger.Printf("beta channel: no prerelease found in %d releases", len(releases))
		return "", fmt.Errorf("no prerelease found")
	}

	// Parse single object response for /releases/latest endpoint
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

func (s *webServer) downloadAndReplaceBinary(binaryPath string) error {
	// Detect platform
	platform := fmt.Sprintf("%s-%s", getOS(), getArch())

	// Construct download URL
	state, _ := config.LoadUpdateState()
	version := state.LatestVersion
	if version == "" {
		return fmt.Errorf("no version to download")
	}

	var downloadURL string
	if strings.Contains(platform, "windows") {
		downloadURL = fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/cando-%s.exe",
			githubRepoOwner, githubRepoName, version, platform)
	} else {
		downloadURL = fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/cando-%s-bin",
			githubRepoOwner, githubRepoName, version, platform)
	}

	s.logger.Printf("downloading from: %s", downloadURL)

	// Download to temp file
	ctx, cancel := context.WithTimeout(context.Background(), updateDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temp file in same directory (for atomic rename)
	dir := filepath.Dir(binaryPath)
	tmpFile, err := os.CreateTemp(dir, "cando-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up on error

	// Copy download to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("failed to chmod: %w", err)
	}

	// Atomic replace: backup old, rename new
	backupPath := binaryPath + ".backup"
	_ = os.Remove(backupPath) // Remove old backup if exists

	if err := os.Rename(binaryPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup: %w", err)
	}

	if err := os.Rename(tmpPath, binaryPath); err != nil {
		// Try to restore backup
		_ = os.Rename(backupPath, binaryPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	s.logger.Printf("binary replaced successfully")
	return nil
}

// checkDirWritable verifies we can create/rename files in the directory.
// We don't need to write to the running binary - we rename it instead.
// This works on both Linux (ETXTBSY prevents writing but not renaming)
// and Windows (running exe can be renamed but not deleted).
func checkDirWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".cando-writetest-*")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}

func compareVersions(v1, v2 string) int {
	// Strip 'v' prefix
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		n1, _ := strconv.Atoi(parts1[i])
		n2, _ := strconv.Atoi(parts2[i])
		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}

	if len(parts1) > len(parts2) {
		return 1
	}
	if len(parts1) < len(parts2) {
		return -1
	}
	return 0
}

func getOS() string {
	return runtime.GOOS
}

func getArch() string {
	return runtime.GOARCH
}

// getManualInstallCommand returns platform-specific install command for manual updates
func getManualInstallCommand() string {
	switch runtime.GOOS {
	case "windows":
		// PowerShell command for Windows - downloads to current directory
		// User needs to move/replace their installed binary manually
		arch := runtime.GOARCH
		if arch == "arm64" {
			return fmt.Sprintf(
				`Invoke-WebRequest -Uri "https://github.com/%s/%s/releases/latest/download/cando-windows-arm64.exe" -OutFile "cando.exe"`,
				githubRepoOwner, githubRepoName)
		}
		return fmt.Sprintf(
			`Invoke-WebRequest -Uri "https://github.com/%s/%s/releases/latest/download/cando-windows-amd64.exe" -OutFile "cando.exe"`,
			githubRepoOwner, githubRepoName)
	default:
		// Bash/curl for Linux and macOS
		return fmt.Sprintf(
			"curl -fsSL https://raw.githubusercontent.com/%s/%s/main/install.sh | bash",
			githubRepoOwner, githubRepoName)
	}
}
