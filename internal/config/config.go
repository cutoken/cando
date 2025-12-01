package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cando/internal/config/migrate"
	"cando/internal/prompts"
	"gopkg.in/yaml.v3"
)

// ProviderModelDefaults holds default models for each provider - single source of truth
type ProviderModelDefaults struct {
	Main    string
	Summary string
	VL      string
}

// ProviderDefaults maps provider keys to their default models
var ProviderDefaults = map[string]ProviderModelDefaults{
	"zai": {
		Main:    "glm-4.6",
		Summary: "glm-4.5-air",
		VL:      "glm-4.5v",
	},
	"openrouter": {
		Main:    "deepseek/deepseek-chat-v3-0324",
		Summary: "qwen/qwen3-30b-a3b-instruct-2507",
		VL:      "qwen/qwen2.5-vl-32b-instruct",
	},
	"mock": {
		Main:    "mock-model",
		Summary: "mock-summary-model",
		VL:      "mock-vl-model",
	},
}

// KnownProviders returns the list of all known provider keys
func KnownProviders() []string {
	return []string{"zai", "openrouter", "mock"}
}

// DefaultConfig returns a config with all defaults set - SINGLE SOURCE OF TRUTH
func DefaultConfig() Config {
	cfg := Config{
		ConfigVersion:         1, // Current version
		Temperature:           0.7,
		ThinkingEnabled:       true,
		ForceThinking:         false,
		ContextProfile:        "memory",
		ContextMessagePercent: 0.02,
		ContextTotalPercent:   0.80,
		ContextProtectRecent:  2,
		WorkspaceRoot:         ".",
		SystemPrompt:          "",
		RequestTimeoutSeconds: 90,
		ShellTimeoutSeconds:   60,
		CompactionPrompt:      DefaultCompactionPrompt,
		ZAIBaseURL:            "https://api.z.ai/api/coding/paas/v4/chat/completions",
		ZAIVisionURL:          "https://api.z.ai/api/coding/paas/v4/chat/completions",
		OpenRouterBaseURL:     "https://openrouter.ai/api/v1",
		OpenRouterVisionURL:   "https://openrouter.ai/api/v1/chat/completions",
		ProviderModels:        make(map[string]string),
		ProviderSummaryModels: make(map[string]string),
		ProviderVLModels:      make(map[string]string),
	}

	// Populate all provider defaults
	for _, p := range KnownProviders() {
		defaults := ProviderDefaults[p]
		cfg.ProviderModels[p] = defaults.Main
		cfg.ProviderSummaryModels[p] = defaults.Summary
		cfg.ProviderVLModels[p] = defaults.VL
	}

	// Set OpenRouter as default provider models
	cfg.Model = ProviderDefaults["openrouter"].Main
	cfg.SummaryModel = ProviderDefaults["openrouter"].Summary
	cfg.VLModel = ProviderDefaults["openrouter"].VL

	return cfg
}

// Config captures the tunable runtime settings for the agent.
const DefaultCompactionPrompt = "Summarize the following text in 20 words or fewer. Return only the summary."

type Config struct {
	ConfigVersion         int               `yaml:"config_version"`
	Model                 string            `yaml:"model"`
	SummaryModel          string            `yaml:"summary_model"`
	VLModel               string            `yaml:"vl_model"`
	BaseURL               string            `yaml:"base_url"`
	Provider              string            `yaml:"provider"`
	ProviderModels        map[string]string `yaml:"provider_models"`
	ProviderSummaryModels map[string]string `yaml:"provider_summary_models"`
	ProviderVLModels      map[string]string `yaml:"provider_vl_models"`
	Temperature           float64           `yaml:"temperature"`
	SystemPrompt          string            `yaml:"system_prompt"`
	RequestTimeoutSeconds int               `yaml:"request_timeout_seconds"`
	ConversationDir       string            `yaml:"conversation_dir"`
	WorkspaceRoot         string            `yaml:"workspace_root"`
	ShellTimeoutSeconds   int               `yaml:"shell_timeout_seconds"`
	ContextProfile        string            `yaml:"context_profile"`
	ZAIBaseURL            string            `yaml:"zai_base_url"`
	ZAIVisionURL          string            `yaml:"zai_vision_url"`
	OpenRouterBaseURL     string            `yaml:"openrouter_base_url"`
	OpenRouterVisionURL   string            `yaml:"openrouter_vision_url"`
	ContextMessagePercent float64           `yaml:"context_message_percent"`
	ContextTotalPercent   float64           `yaml:"context_conversation_percent"`
	ContextProtectRecent  int               `yaml:"context_protect_recent"`
	MemoryStorePath       string            `yaml:"memory_store_path"`
	HistoryPath           string            `yaml:"history_path"`
	ThinkingEnabled       bool              `yaml:"thinking_enabled"`
	ForceThinking         bool              `yaml:"force_thinking"`
	CompactionPrompt      string            `yaml:"compaction_summary_prompt"`
	OpenRouterFreeMode    bool              `yaml:"openrouter_free_mode"`
	AnalyticsEnabled      *bool             `yaml:"analytics_enabled,omitempty"` // nil = default true
}

// IsAnalyticsEnabled returns true if analytics is enabled (default: true)
func (c Config) IsAnalyticsEnabled() bool {
	if c.AnalyticsEnabled == nil {
		return true // default on
	}
	return *c.AnalyticsEnabled
}

// EnsureDefaultConfig creates config.yaml with provider-appropriate defaults if it doesn't exist
func EnsureDefaultConfig(provider string) error {
	configDir := GetConfigDir()
	configPath := filepath.Join(configDir, "config.yaml")

	// If config already exists, ensure all providers have defaults
	if _, err := os.Stat(configPath); err == nil {
		// Config exists, but might be missing some provider defaults
		return EnsureAllProviderDefaults(configPath)
	}

	// Create .cando directory if needed
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Start with defaults from single source of truth
	cfg := DefaultConfig()

	// Override with provider-specific settings
	provider = strings.ToLower(provider)
	if defaults, ok := ProviderDefaults[provider]; ok {
		cfg.Model = defaults.Main
		cfg.SummaryModel = defaults.Summary
		cfg.VLModel = defaults.VL
	} else if provider != "" {
		// Unknown provider - add it with OpenRouter defaults
		orDefaults := ProviderDefaults["openrouter"]
		cfg.ProviderModels[provider] = orDefaults.Main
		cfg.ProviderSummaryModels[provider] = orDefaults.Summary
		cfg.ProviderVLModels[provider] = orDefaults.VL
	}

	// No provider-specific overrides - everything comes from DefaultConfig

	// Write config file
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// EnsureAllProviderDefaults ensures all provider maps have default values for all known providers.
// Only writes to disk if changes were actually made.
func EnsureAllProviderDefaults(configPath string) error {
	// Load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	// Track if any changes were made
	var changes []string

	// Ensure thinking is enabled by default for existing configs
	// This handles upgrades from older versions without the field
	if !cfg.ThinkingEnabled && !cfg.ForceThinking {
		cfg.ThinkingEnabled = true
		changes = append(changes, "enabled thinking mode (default)")
	}

	// Ensure provider URLs are set
	defaultCfg := DefaultConfig()
	if cfg.ZAIBaseURL == "" {
		cfg.ZAIBaseURL = defaultCfg.ZAIBaseURL
		changes = append(changes, fmt.Sprintf("zai_base_url=%s", cfg.ZAIBaseURL))
	}
	if cfg.ZAIVisionURL == "" {
		cfg.ZAIVisionURL = defaultCfg.ZAIVisionURL
		changes = append(changes, fmt.Sprintf("zai_vision_url=%s", cfg.ZAIVisionURL))
	}
	if cfg.OpenRouterBaseURL == "" {
		cfg.OpenRouterBaseURL = defaultCfg.OpenRouterBaseURL
		changes = append(changes, fmt.Sprintf("openrouter_base_url=%s", cfg.OpenRouterBaseURL))
	}
	if cfg.OpenRouterVisionURL == "" {
		cfg.OpenRouterVisionURL = defaultCfg.OpenRouterVisionURL
		changes = append(changes, fmt.Sprintf("openrouter_vision_url=%s", cfg.OpenRouterVisionURL))
	}

	// Initialize maps if nil
	if cfg.ProviderModels == nil {
		cfg.ProviderModels = make(map[string]string)
	}
	if cfg.ProviderSummaryModels == nil {
		cfg.ProviderSummaryModels = make(map[string]string)
	}
	if cfg.ProviderVLModels == nil {
		cfg.ProviderVLModels = make(map[string]string)
	}

	// Ensure all providers have defaults using the ProviderDefaults map
	for _, provider := range KnownProviders() {
		defaults, ok := ProviderDefaults[provider]
		if !ok {
			continue
		}

		// Main models
		if cfg.ProviderModels[provider] == "" {
			cfg.ProviderModels[provider] = defaults.Main
			changes = append(changes, fmt.Sprintf("%s.main=%s", provider, defaults.Main))
		}

		// Summary models
		if cfg.ProviderSummaryModels[provider] == "" {
			cfg.ProviderSummaryModels[provider] = defaults.Summary
			changes = append(changes, fmt.Sprintf("%s.summary=%s", provider, defaults.Summary))
		}

		// VL models
		if cfg.ProviderVLModels[provider] == "" {
			cfg.ProviderVLModels[provider] = defaults.VL
			changes = append(changes, fmt.Sprintf("%s.vl=%s", provider, defaults.VL))
		}
	}

	// Only write if changes were made
	if len(changes) == 0 {
		return nil
	}

	log.Printf("Config: adding missing provider defaults: %v", changes)

	updatedData, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal updated config: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("write updated config: %w", err)
	}

	return nil
}

// LoadUserConfig loads configuration from ~/.cando/config.yaml
// Checks CANDO_CONFIG_PATH environment variable first.
// If the file doesn't exist, returns defaults
func LoadUserConfig() (Config, error) {
	configPath := os.Getenv("CANDO_CONFIG_PATH")
	if configPath == "" {
		configPath = filepath.Join(GetConfigDir(), "config.yaml")
	}

	// Run migrations first (if config exists)
	if _, err := os.Stat(configPath); err == nil {
		if err := migrate.MigrateConfig(configPath); err != nil {
			log.Printf("Warning: config migration failed: %v", err)
			// Continue with unmigrated config
		}
	}

	// If file doesn't exist, create it with defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create config with defaults from single source of truth
		cfg := DefaultConfig()

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return Config{}, fmt.Errorf("create config dir: %w", err)
		}

		// Write the default config
		data, err := yaml.Marshal(&cfg)
		if err != nil {
			return Config{}, fmt.Errorf("marshal default config: %w", err)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return Config{}, fmt.Errorf("write default config: %w", err)
		}

		// Apply computed values before returning
		cfg.applyComputedPaths()
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyComputedPaths()
	cfg.cleanSystemPrompt()
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Load reads the YAML configuration from disk and injects sane defaults.
// Deprecated: Use LoadUserConfig for user config
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	cfg.applyComputedPaths()
	cfg.cleanSystemPrompt()
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// applyComputedPaths sets computed paths based on workspace root and config dir
// This ONLY sets paths that are derived from other values, never user preferences
func (c *Config) applyComputedPaths() {
	// Only set computed paths if not already set
	if c.ConversationDir == "" {
		c.ConversationDir = filepath.Join(GetConfigDir(), "conversations")
	}
	// Only set workspace-dependent paths if we have a workspace root
	// For web UI mode without explicit workspace, these are set per-workspace later
	if c.MemoryStorePath == "" && c.WorkspaceRoot != "" {
		c.MemoryStorePath = filepath.Join(c.WorkspaceRoot, "memory.db")
	}
	if c.HistoryPath == "" && c.WorkspaceRoot != "" {
		c.HistoryPath = filepath.Join(c.WorkspaceRoot, ".cando_history")
	}
}

// cleanSystemPrompt removes the base prompt and environment context if present,
// ensuring only the user's custom portion is stored in the config.
func (c *Config) cleanSystemPrompt() {
	c.SystemPrompt = prompts.ExtractUserPortion(c.SystemPrompt)
}

func (c Config) validate() error {
	if c.ContextMessagePercent <= 0 || c.ContextMessagePercent > 0.10 {
		return fmt.Errorf("context_message_percent must be between 0 and 0.10 (0-10%%)")
	}
	if c.ContextTotalPercent <= 0 || c.ContextTotalPercent > 0.80 {
		return fmt.Errorf("context_conversation_percent must be between 0 and 0.80 (0-80%%)")
	}
	// Logical consistency check: per-message threshold cannot exceed total conversation threshold
	if c.ContextMessagePercent > c.ContextTotalPercent {
		return fmt.Errorf("context_message_percent (%f) cannot exceed context_conversation_percent (%f)", c.ContextMessagePercent, c.ContextTotalPercent)
	}
	if c.ContextProtectRecent < 0 {
		return fmt.Errorf("context_protect_recent must be >= 0")
	}
	// Temperature validation (typical LLM range is 0-2.0)
	if c.Temperature < 0 || c.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0 and 2.0 (got %f)", c.Temperature)
	}
	// Timeout sanity checks
	if c.RequestTimeoutSeconds > 600 {
		return fmt.Errorf("request_timeout_seconds cannot exceed 600 (10 minutes)")
	}
	if c.ShellTimeoutSeconds > 600 {
		return fmt.Errorf("shell_timeout_seconds cannot exceed 600 (10 minutes)")
	}
	if strings.TrimSpace(c.MemoryStorePath) == "" {
		return fmt.Errorf("memory_store_path must be set")
	}
	if strings.TrimSpace(c.HistoryPath) == "" {
		return fmt.Errorf("history_path must be set")
	}
	if strings.TrimSpace(c.SummaryModel) == "" {
		return fmt.Errorf("summary_model must be set")
	}
	return nil
}

// RequestTimeout turns the integer value into a duration for HTTP clients.
func (c Config) RequestTimeout() time.Duration {
	return time.Duration(c.RequestTimeoutSeconds) * time.Second
}

// ShellTimeout exposes the configured duration for sandboxed shell commands.
func (c Config) ShellTimeout() time.Duration {
	return time.Duration(c.ShellTimeoutSeconds) * time.Second
}

// OverrideWorkspaceRoot swaps the workspace root at runtime and rebases dependent paths.
func (c *Config) OverrideWorkspaceRoot(root string) {
	if c == nil {
		return
	}
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		return
	}
	oldRoot := c.WorkspaceRoot
	c.WorkspaceRoot = trimmed
	c.rebasePath(&c.MemoryStorePath, oldRoot, trimmed)
	c.rebasePath(&c.HistoryPath, oldRoot, trimmed)
}

func (c *Config) rebasePath(target *string, oldRoot, newRoot string) {
	if target == nil {
		return
	}
	val := strings.TrimSpace(*target)
	if val == "" {
		return
	}
	oldAbs := absPath(oldRoot)
	newAbs := absPath(newRoot)
	pathVal := val
	if filepath.IsAbs(pathVal) {
		if oldAbs == "" {
			return
		}
		rel, err := filepath.Rel(oldAbs, pathVal)
		if err != nil || strings.HasPrefix(rel, "..") {
			return
		}
		pathVal = rel
	}
	if newAbs == "" {
		newAbs = "."
	}
	*target = filepath.Join(newAbs, pathVal)
}

func absPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}

func GetConfigDir() string {
	if configDir := os.Getenv("CANDO_CONFIG_DIR"); configDir != "" {
		return configDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".cando"
	}

	// Check if running as beta version
	execName := filepath.Base(os.Args[0])
	if strings.Contains(execName, "beta") {
		return filepath.Join(home, ".cando-beta")
	}

	return filepath.Join(home, ".cando")
}

// ModelFor returns the configured model for the given provider key, falling back to provider-appropriate defaults.
func (c Config) ModelFor(provider string) string {
	provider = strings.ToLower(provider)

	if len(c.ProviderModels) > 0 {
		if model := strings.TrimSpace(c.ProviderModels[provider]); model != "" {
			return model
		}
	}

	if defaults, ok := ProviderDefaults[provider]; ok {
		return defaults.Main
	}
	return c.Model
}

// SummaryModelFor returns the configured summary model for the given provider key, falling back to provider-appropriate defaults.
func (c Config) SummaryModelFor(provider string) string {
	provider = strings.ToLower(provider)

	if len(c.ProviderSummaryModels) > 0 {
		if model := strings.TrimSpace(c.ProviderSummaryModels[provider]); model != "" {
			return model
		}
	}

	if defaults, ok := ProviderDefaults[provider]; ok {
		return defaults.Summary
	}
	return c.SummaryModel
}

// VLModelFor returns the appropriate VL (Vision Language) model for a provider
func (c Config) VLModelFor(provider string) string {
	provider = strings.ToLower(provider)

	if len(c.ProviderVLModels) > 0 {
		if model := strings.TrimSpace(c.ProviderVLModels[provider]); model != "" {
			return model
		}
	}

	if defaults, ok := ProviderDefaults[provider]; ok {
		return defaults.VL
	}
	if model := strings.TrimSpace(c.VLModel); model != "" {
		return model
	}
	return ProviderDefaults["openrouter"].VL
}

// CalculateMessageThreshold returns the absolute character threshold for message compaction
// based on the configured percentage and model context length.
// Uses 3:1 character-to-token ratio (conservative estimate).
func (c Config) CalculateMessageThreshold(provider, model string) int {
	tokens := GetModelContextLength(provider, model)
	chars := tokens * 3
	threshold := int(float64(chars) * c.ContextMessagePercent)
	if threshold <= 0 {
		threshold = 1000 // Minimum fallback
	}
	return threshold
}

// CalculateConversationThreshold returns the absolute character threshold for conversation compaction
// based on the configured percentage and model context length.
// Uses 3:1 character-to-token ratio (conservative estimate).
func (c Config) CalculateConversationThreshold(provider, model string) int {
	tokens := GetModelContextLength(provider, model)
	chars := tokens * 3
	threshold := int(float64(chars) * c.ContextTotalPercent)
	if threshold <= 0 {
		threshold = 10000 // Minimum fallback
	}
	return threshold
}

// Save writes the config to the user's config file
func Save(c Config) error {
	configPath := os.Getenv("CANDO_CONFIG_PATH")
	if configPath == "" {
		configPath = filepath.Join(GetConfigDir(), "config.yaml")
	}

	// Clear runtime-calculated paths before saving
	// These are set dynamically based on workspace and shouldn't be persisted
	saveConfig := c
	saveConfig.ConversationDir = ""
	saveConfig.MemoryStorePath = ""
	saveConfig.HistoryPath = ""

	data, err := yaml.Marshal(saveConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
