package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cando/internal/prompts"
	"gopkg.in/yaml.v3"
)

// Provider-specific default model constants
const (
	// Main model defaults
	DefaultZAIModel        = "glm-4.6"
	DefaultOpenRouterModel = "deepseek/deepseek-chat-v3-0324"
	DefaultMockModel       = "mock-model"
	
	// Summary model defaults  
	DefaultZAISummaryModel        = "glm-4.5-air"
	DefaultOpenRouterSummaryModel = "qwen/qwen3-30b-a3b-instruct-2507"
	DefaultMockSummaryModel       = "mock-summary-model"
	
	// VL (Vision Language) model defaults
	DefaultZAIVLModel        = "glm-4.5v"
	DefaultOpenRouterVLModel = "qwen/qwen2.5-vl-32b-instruct"
	DefaultMockVLModel       = "mock-vl-model"
)

// Config captures the tunable runtime settings for the agent.
const DefaultCompactionPrompt = "Summarize the following text in 20 words or fewer. Return only the summary."

type Config struct {
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
	ContextMessagePercent float64           `yaml:"context_message_percent"`
	ContextTotalPercent   float64           `yaml:"context_conversation_percent"`
	ContextProtectRecent  int               `yaml:"context_protect_recent"`
	MemoryStorePath       string            `yaml:"memory_store_path"`
	HistoryPath           string            `yaml:"history_path"`
	ThinkingEnabled       bool              `yaml:"thinking_enabled"`
	ForceThinking         bool              `yaml:"force_thinking"`
	CompactionPrompt      string            `yaml:"compaction_summary_prompt"`
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

	cfg := Config{}
	switch strings.ToLower(provider) {
	case "zai":
		cfg.Model = DefaultZAIModel
		cfg.ProviderModels = map[string]string{
			"zai":        DefaultZAIModel,
			"openrouter": DefaultOpenRouterModel,
			"mock":       DefaultMockModel,
		}
		cfg.SummaryModel = DefaultZAISummaryModel
		cfg.ProviderSummaryModels = map[string]string{
			"zai":        DefaultZAISummaryModel,
			"openrouter": DefaultOpenRouterSummaryModel,
			"mock":       DefaultMockSummaryModel,
		}
		cfg.VLModel = DefaultZAIVLModel
		cfg.ProviderVLModels = map[string]string{
			"zai":        DefaultZAIVLModel,
			"openrouter": DefaultOpenRouterVLModel,
			"mock":       DefaultMockVLModel,
		}
		cfg.ZAIBaseURL = "https://api.z.ai/api/coding/paas/v4/chat/completions"
	case "openrouter":
		cfg.Model = DefaultOpenRouterModel
		cfg.ProviderModels = map[string]string{
			"zai":        DefaultZAIModel,
			"openrouter": DefaultOpenRouterModel,
			"mock":       DefaultMockModel,
		}
		cfg.SummaryModel = DefaultOpenRouterSummaryModel
		cfg.ProviderSummaryModels = map[string]string{
			"zai":        DefaultZAISummaryModel,
			"openrouter": DefaultOpenRouterSummaryModel,
			"mock":       DefaultMockSummaryModel,
		}
		cfg.VLModel = DefaultOpenRouterVLModel
		cfg.ProviderVLModels = map[string]string{
			"zai":        DefaultZAIVLModel,
			"openrouter": DefaultOpenRouterVLModel,
			"mock":       DefaultMockVLModel,
		}
	default:
		// Use general defaults
		cfg.Model = DefaultOpenRouterModel
	}

	// Apply standard defaults for other fields
	cfg.Temperature = 0.7
	cfg.ThinkingEnabled = true
	cfg.ForceThinking = true
	cfg.ContextProfile = "memory"
	cfg.ContextMessagePercent = 0.02 // 2% of model context
	cfg.ContextTotalPercent = 0.80   // 80% of model context
	cfg.ContextProtectRecent = 2
	cfg.CompactionPrompt = DefaultCompactionPrompt
	cfg.WorkspaceRoot = "."
	cfg.SystemPrompt = ""

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

// EnsureAllProviderDefaults ensures all provider maps have default values for all known providers
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

	// Ensure all providers have defaults in all maps
	knownProviders := []string{"zai", "openrouter", "mock"}
	
	for _, provider := range knownProviders {
		// Main models
		if cfg.ProviderModels[provider] == "" {
			switch provider {
			case "zai":
				cfg.ProviderModels[provider] = DefaultZAIModel
			case "openrouter":
				cfg.ProviderModels[provider] = DefaultOpenRouterModel
			case "mock":
				cfg.ProviderModels[provider] = DefaultMockModel
			}
		}

		// Summary models
		if cfg.ProviderSummaryModels[provider] == "" {
			switch provider {
			case "zai":
				cfg.ProviderSummaryModels[provider] = DefaultZAISummaryModel
			case "openrouter":
				cfg.ProviderSummaryModels[provider] = DefaultOpenRouterSummaryModel
			case "mock":
				cfg.ProviderSummaryModels[provider] = DefaultMockSummaryModel
			}
		}

		// VL models
		if cfg.ProviderVLModels[provider] == "" {
			switch provider {
			case "zai":
				cfg.ProviderVLModels[provider] = DefaultZAIVLModel
			case "openrouter":
				cfg.ProviderVLModels[provider] = DefaultOpenRouterVLModel
			case "mock":
				cfg.ProviderVLModels[provider] = DefaultMockVLModel
			}
		}
	}

	// Write back the updated config
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

	// If file doesn't exist, return defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := Config{}
		cfg.applyDefaults()
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

	cfg.applyDefaults()
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
	cfg.applyDefaults()
	cfg.cleanSystemPrompt()
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// applyDefaults fills in optional values to keep the YAML file concise.
func (c *Config) applyDefaults() {
	if c.BaseURL == "" {
		c.BaseURL = "https://openrouter.ai/api/v1"
	}
	// Note: Model and SummaryModel defaults are handled by EnsureDefaultConfig() 
	// and provider-aware fallbacks in ModelFor()/SummaryModelFor() to avoid conflicts
	if c.Temperature == 0 {
		c.Temperature = 0.2
	}
	if c.RequestTimeoutSeconds <= 0 {
		c.RequestTimeoutSeconds = 90
	}
	if c.ConversationDir == "" {
		c.ConversationDir = filepath.Join(GetConfigDir(), "conversations")
	}
	if c.WorkspaceRoot == "" {
		c.WorkspaceRoot = "."
	}
	if c.ShellTimeoutSeconds <= 0 {
		c.ShellTimeoutSeconds = 60
	}
	if c.ContextProfile == "" {
		c.ContextProfile = "default"
	}
	if c.ZAIBaseURL == "" {
		c.ZAIBaseURL = "https://api.z.ai/api/coding/paas/v4/chat/completions"
	}
	if c.ContextMessagePercent <= 0 {
		c.ContextMessagePercent = 0.02 // 2% default
	}
	if c.ContextTotalPercent <= 0 {
		c.ContextTotalPercent = 0.80 // 80% default
	}
	if c.MemoryStorePath == "" {
		root := c.WorkspaceRoot
		if root == "" {
			root = "."
		}
		c.MemoryStorePath = filepath.Join(root, "memory.db")
	}
	if c.HistoryPath == "" {
		root := c.WorkspaceRoot
		if root == "" {
			root = "."
		}
		c.HistoryPath = filepath.Join(root, ".cando_history")
	}
	if strings.TrimSpace(c.CompactionPrompt) == "" {
		c.CompactionPrompt = DefaultCompactionPrompt
	}
	if strings.TrimSpace(c.SummaryModel) == "" {
		c.SummaryModel = "glm-4.5-air"
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
	
	switch provider {
	case "zai":
		return DefaultZAIModel
	case "openrouter":
		return DefaultOpenRouterModel
	case "mock":
		return DefaultMockModel
	default:
		return c.Model
	}
}

// SummaryModelFor returns the configured summary model for the given provider key, falling back to provider-appropriate defaults.
func (c Config) SummaryModelFor(provider string) string {
	provider = strings.ToLower(provider)
	
	if len(c.ProviderSummaryModels) > 0 {
		if model := strings.TrimSpace(c.ProviderSummaryModels[provider]); model != "" {
			return model
		}
	}
	
	switch provider {
	case "zai":
		return DefaultZAISummaryModel
	case "openrouter":
		return DefaultOpenRouterSummaryModel
	case "mock":
		return DefaultMockSummaryModel
	default:
		return c.SummaryModel
	}
}

// VLModelFor returns the appropriate VL (Vision Language) model for a provider
func (c Config) VLModelFor(provider string) string {
	provider = strings.ToLower(provider)
	
	if len(c.ProviderVLModels) > 0 {
		if model := strings.TrimSpace(c.ProviderVLModels[provider]); model != "" {
			return model
		}
	}
	
	switch provider {
	case "zai":
		return DefaultZAIVLModel
	case "openrouter":
		return DefaultOpenRouterVLModel
	case "mock":
		return DefaultMockVLModel
	default:
		if model := strings.TrimSpace(c.VLModel); model != "" {
			return model
		}
		return DefaultOpenRouterVLModel
	}
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

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
