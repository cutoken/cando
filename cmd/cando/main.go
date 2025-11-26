package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"cando/internal/agent"
	"cando/internal/config"
	"cando/internal/contextprofile"
	"cando/internal/credentials"
	"cando/internal/llm"
	mockclient "cando/internal/llm/mockclient"
	"cando/internal/openrouter"
	"cando/internal/prompts"
	"cando/internal/state"
	"cando/internal/tooling"
	"cando/internal/zai"
)

// Version is set via -ldflags during build
var Version = "dev"

func main() {
	// Parse flags
	var (
		sandboxPath  = flag.String("sandbox", "", "Override workspace root/sandbox directory")
		resumeKey    = flag.String("resume", "", "Resume an existing session key")
		listSessions = flag.Bool("list-sessions", false, "List stored sessions for this workspace and exit")
		port         = flag.Int("port", 0, "Port for web UI (default: auto-select starting at 3737)")
		promptFlag   = flag.String("p", "", "Execute a single prompt and exit (non-interactive mode)")
		setupFlag    = flag.Bool("setup", false, "Run credential setup wizard")
		versionFlag  = flag.Bool("version", false, "Print version and exit")
	)
	flag.StringVar(promptFlag, "prompt", "", "Execute a single prompt and exit (non-interactive mode)")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Cando version %s\n", Version)
		return
	}

	// Handle credential setup
	if *setupFlag {
		credManager, err := credentials.NewManager()
		if err != nil {
			log.Fatalf("Failed to initialize credential manager: %v", err)
		}
		if err := credentials.SetupMenu(credManager); err != nil {
			log.Fatalf("Setup failed: %v", err)
		}
		return
	}

	// Load credential manager
	credManager, err := credentials.NewManager()
	if err != nil {
		log.Fatalf("Failed to initialize credential manager: %v", err)
	}

	creds, err := credManager.Load()
	if err != nil {
		log.Fatalf("Failed to load credentials: %v", err)
	}

	// Ensure config exists with provider-appropriate defaults
	if creds.HasAnyProvider() {
		if err := config.EnsureDefaultConfig(creds.DefaultProvider); err != nil {
			log.Fatalf("Failed to ensure default config: %v", err)
		}
	}

	// Load user configuration
	cfg, err := config.LoadUserConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override workspace if specified
	if sandbox := strings.TrimSpace(*sandboxPath); sandbox != "" {
		cfg.OverrideWorkspaceRoot(sandbox)
	}

	// Determine provider from credentials (may be empty for first-run)
	activeProvider := strings.ToLower(creds.DefaultProvider)
	hasCredentials := creds.HasAnyProvider()

	// Set up workspace
	root := cfg.WorkspaceRoot
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		log.Fatalf("Failed to resolve workspace root: %v", err)
	}
	if err := os.MkdirAll(absRoot, 0o755); err != nil {
		log.Fatalf("Failed to create workspace root: %v", err)
	}

	dataRoot, err := projectStorageRoot(absRoot)
	if err != nil {
		log.Fatalf("Failed to determine project storage root: %v", err)
	}
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		log.Fatalf("Failed to create project storage root: %v", err)
	}

	cfg.ConversationDir = filepath.Join(dataRoot, "conversations")
	cfg.MemoryStorePath = filepath.Join(dataRoot, "memory.db")
	cfg.HistoryPath = filepath.Join(dataRoot, ".history")

	prompts.SetMetadata(buildEnvironmentMetadata(absRoot))

	// Set up logging
	logPath := filepath.Join(dataRoot, "cando.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	logger := log.New(logFile, "cando ", log.LstdFlags|log.Lmicroseconds)

	// Build provider registrations using credentials or mock client for tests
	var client llm.Client
	mockMode := os.Getenv("CANDO_MOCK_LLM") == "1"
	if mockMode {
		logger.Println("CANDO_MOCK_LLM=1 detected; using mock LLM client")
		client = mockclient.New()
		hasCredentials = true
		activeProvider = "mock"
	} else if hasCredentials {
		providerRegs := make([]agent.ProviderRegistration, 0, 2)

		if creds.IsConfigured("zai") {
			if reg, err := buildZAIRegistration(cfg, creds.GetAPIKey("zai"), logger); err != nil {
				if activeProvider == "zai" {
					log.Fatalf("Failed to init Z.AI provider: %v", err)
				}
				logger.Printf("Warning: Z.AI provider init failed: %v", err)
			} else if reg != nil {
				providerRegs = append(providerRegs, *reg)
			}
		}

		if creds.IsConfigured("openrouter") {
			if reg, err := buildOpenRouterRegistration(cfg, creds.GetAPIKey("openrouter"), logger); err != nil {
				if activeProvider == "openrouter" {
					log.Fatalf("Failed to init OpenRouter provider: %v", err)
				}
				logger.Printf("Warning: OpenRouter provider init failed: %v", err)
			} else if reg != nil {
				providerRegs = append(providerRegs, *reg)
			}
		}

		// Select client
		if len(providerRegs) == 0 {
			log.Fatal("No providers configured. Run: cando --setup")
		} else {
			// Always wrap in multi-provider client for consistent UI exposure
			multi, err := agent.NewMultiProviderClient(activeProvider, providerRegs)
			if err != nil {
				log.Fatalf("Failed to init multi-provider client: %v", err)
			}
			client = multi
			if len(providerRegs) == 1 {
				logger.Printf("Using %s provider", strings.ToUpper(providerRegs[0].Option.Key))
			} else {
				logger.Printf("Providers available: %s (default=%s)", providerLabels(providerRegs), activeProvider)
			}
		}
	} else {
		logger.Println("No credentials configured - web UI will show onboarding")
	}

	// Initialize state manager
	combinedPrompt := prompts.Combine(cfg.SystemPrompt)
	states, err := state.NewManager(combinedPrompt, cfg.ConversationDir, logger)
	if err != nil {
		log.Fatalf("Failed to init state manager: %v", err)
	}

	// Handle list-sessions
	if *listSessions {
		keys := states.ListKeys()
		printSessionList(keys)
		return
	}

	// Set up tools
	toolOpts := tooling.Options{
		WorkspaceRoot: absRoot,
		ShellTimeout:  cfg.ShellTimeout(),
		PlanPath:      filepath.Join(dataRoot, "plan.json"),
		BinDir:        globalBinDir(),
		ExternalData:  true,
		ProcessDir:    filepath.Join(dataRoot, "processes"),
		CredManager:   credManager,
	}
	baseTools := tooling.DefaultTools(toolOpts)

	// Initialize context profile
	// Force default profile if no credentials (memory profile needs LLM client)
	profileType := cfg.ContextProfile
	if !hasCredentials {
		profileType = "default"
	}
	// Determine active model for compaction threshold calculations
	profileModel := cfg.Model
	if hasCredentials {
		profileModel = cfg.ModelFor(activeProvider)
	}
	profile, err := contextprofile.New(profileType, contextprofile.Dependencies{
		Client:   client,
		Logger:   logger,
		Config:   cfg,
		Provider: activeProvider,
		Model:    profileModel,
	})
	if err != nil {
		log.Fatalf("Failed to init context profile: %v", err)
	}
	allTools := append(baseTools, profile.Tools()...)
	tools := tooling.NewRegistry(allTools...)

	// Set tool definitions in profile for compaction calculations
	if setter, ok := profile.(interface {
		SetToolDefinitions([]tooling.ToolDefinition)
	}); ok {
		setter.SetToolDefinitions(tools.Definitions())
	}

	// Create agent with provider builders for dynamic reloading
	providerBuilders := map[string]agent.ProviderBuilder{
		"zai":        buildZAIRegistration,
		"openrouter": buildOpenRouterRegistration,
	}

	agentInstance := agent.New(client, cfg, "", states, profile, tools, logger, credManager, agent.Options{
		ResumeKey:        strings.TrimSpace(*resumeKey),
		WorkspaceRoot:    absRoot,
		ProviderBuilders: providerBuilders,
		ActiveProvider:   activeProvider,
		ProfileModel:     profileModel,
	}, toolOpts)

	// Handle one-shot prompt mode
	if *promptFlag != "" {
		if err := runOneShotPrompt(agentInstance, *promptFlag); err != nil {
			log.Fatalf("Prompt failed: %v", err)
		}
		return
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Println("Received shutdown signal, gracefully stopping...")
		cancel()
	}()

	// Determine port
	listenPort := 3737
	if portEnv := os.Getenv("CANDO_PORT"); portEnv != "" {
		if port, err := strconv.Atoi(portEnv); err == nil && port > 0 {
			listenPort = port
		}
	} else if os.Getenv("DEV_MODE") == "true" {
		listenPort = 4000
	}
	if *port > 0 {
		listenPort = *port
	}

	// Find available port
	listenAddr := findAvailablePort(listenPort)

	// Start web UI
	fmt.Printf("Starting Cando...\n")
	fmt.Printf("→ Web UI: http://%s\n", listenAddr)
	fmt.Println()

	// Auto-open browser (skip in dev mode to avoid repeated launches)
	if os.Getenv("DEV_MODE") == "" {
		go openBrowser("http://" + listenAddr)
	}

	if err := agentInstance.RunWeb(ctx, listenAddr); err != nil {
		log.Fatalf("Web UI failed: %v", err)
	}
}

func runOneShotPrompt(agentInstance *agent.Agent, prompt string) error {
	ctx := context.Background()
	return agentInstance.RunOneShot(ctx, prompt)
}

func findAvailablePort(startPort int) string {
	for port := startPort; port < startPort+100; port++ {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return addr
		}
	}
	// Fallback to let OS pick
	return "127.0.0.1:0"
}

func projectStorageRoot(workspace string) (string, error) {
	slug := projectSlug(workspace)
	return filepath.Join(config.GetConfigDir(), "projects", slug), nil
}

func projectSlug(path string) string {
	clean := filepath.Clean(path)
	base := sanitizeSlug(filepath.Base(clean))
	if base == "" {
		base = "workspace"
	}
	sum := sha1.Sum([]byte(clean))
	hash := hex.EncodeToString(sum[:8])
	return fmt.Sprintf("%s-%s", base, hash)
}

func sanitizeSlug(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		case r == '-' || r == '_':
			b.WriteRune('-')
		case unicode.IsSpace(r):
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func printSessionList(keys []string) {
	if len(keys) == 0 {
		fmt.Println("No stored sessions for this workspace yet.")
		return
	}
	fmt.Printf("Stored sessions (%d):\n", len(keys))
	for i, key := range keys {
		fmt.Printf("  %d) %s\n", i+1, key)
	}
}

func buildEnvironmentMetadata(workspace string) string {
	now := time.Now()
	zoneName, offset := now.Zone()
	if strings.TrimSpace(zoneName) == "" {
		zoneName = "Local"
	}
	lines := []string{
		fmt.Sprintf("- OS: %s (%s)", runtime.GOOS, runtime.GOARCH),
	}
	if shell := detectShell(); shell != "" {
		lines = append(lines, fmt.Sprintf("- Shell: %s", shell))
	}
	lines = append(lines, fmt.Sprintf("- Date: %s", now.Format("2006-01-02")))
	lines = append(lines, fmt.Sprintf("- Timezone: %s (UTC%s)", zoneName, formatUTCOffset(offset)))
	if locale := detectLocale(); locale != "" {
		lines = append(lines, fmt.Sprintf("- System Language: %s", locale))
	}
	if workspace != "" {
		lines = append(lines, fmt.Sprintf("- Workspace Root: %s", workspace))
	}
	if Version != "" {
		lines = append(lines, fmt.Sprintf("- Cando Version: %s", Version))
	}
	return strings.Join(lines, "\n")
}

func detectShell() string {
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		return shell
	}
	if shell := strings.TrimSpace(os.Getenv("COMSPEC")); shell != "" {
		return shell
	}
	return ""
}

func detectLocale() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if val := strings.TrimSpace(os.Getenv(key)); val != "" {
			return val
		}
	}
	return ""
}

func formatUTCOffset(offsetSeconds int) string {
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}

func buildZAIRegistration(cfg config.Config, apiKey string, logger *log.Logger) (*agent.ProviderRegistration, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Z.AI API key not configured")
	}
	base := cfg.ZAIBaseURL
	if base == "" {
		base = "https://api.z.ai/api"
	}
	client := zai.NewClient(base, apiKey, cfg.RequestTimeout(), logger)
	model := cfg.ModelFor("zai")
	logger.Printf("Z.AI provider ready (model %s)", model)
	return &agent.ProviderRegistration{
		Option: agent.ProviderOption{
			Key:    "zai",
			Label:  fmt.Sprintf("GLM · %s", model),
			Model:  model,
			Source: "zai",
		},
		Client: client,
	}, nil
}

func buildOpenRouterRegistration(cfg config.Config, apiKey string, logger *log.Logger) (*agent.ProviderRegistration, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key not configured")
	}
	endpoint := cfg.BaseURL
	if endpoint == "" {
		endpoint = "https://openrouter.ai/api/v1"
	}
	client := openrouter.NewClient(endpoint, apiKey, cfg.RequestTimeout(), logger)
	model := cfg.ModelFor("openrouter")
	if model == "" {
		model = cfg.Model
	}
	logger.Printf("OpenRouter provider ready (model %s)", model)
	return &agent.ProviderRegistration{
		Option: agent.ProviderOption{
			Key:    "openrouter",
			Label:  fmt.Sprintf("OpenRouter · %s", model),
			Model:  model,
			Source: "openrouter",
		},
		Client: client,
	}, nil
}

func providerLabels(regs []agent.ProviderRegistration) string {
	if len(regs) == 0 {
		return ""
	}
	names := make([]string, 0, len(regs))
	for _, reg := range regs {
		names = append(names, reg.Option.Key)
	}
	return strings.Join(names, ", ")
}

func globalBinDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cando", "bin")
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler"}
	case "darwin":
		cmd = "open"
	default:
		return
	}

	args = append(args, url)
	exec.Command(cmd, args...).Start()
}
