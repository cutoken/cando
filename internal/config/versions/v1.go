package versions

// ConfigV1 is the first versioned config with proper defaults
// Changes from v0:
// - Added ConfigVersion field
// - Fixed default values (thinking=true, context_profile=memory)
// - Removed BaseURL from defaults (provider-specific)
type ConfigV1 struct {
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
	ContextMessagePercent float64           `yaml:"context_message_percent"`
	ContextTotalPercent   float64           `yaml:"context_conversation_percent"`
	ContextProtectRecent  int               `yaml:"context_protect_recent"`
	MemoryStorePath       string            `yaml:"memory_store_path"`
	HistoryPath           string            `yaml:"history_path"`
	ThinkingEnabled       bool              `yaml:"thinking_enabled"`
	ForceThinking         bool              `yaml:"force_thinking"`
	CompactionPrompt      string            `yaml:"compaction_summary_prompt"`
	OpenRouterFreeMode    bool              `yaml:"openrouter_free_mode"`
	AnalyticsEnabled      *bool             `yaml:"analytics_enabled,omitempty"`
}
