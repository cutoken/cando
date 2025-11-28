package migrate

import (
	"cando/internal/config/versions"
	"gopkg.in/yaml.v3"
)

// MigrationV0toV1 migrates from unversioned config to v1
type MigrationV0toV1 struct{}

func (m *MigrationV0toV1) FromVersion() int { return Version0 }
func (m *MigrationV0toV1) ToVersion() int   { return Version1 }
func (m *MigrationV0toV1) Description() string {
	return "Add versioning, fix defaults (thinking=true, context=memory)"
}

func (m *MigrationV0toV1) Migrate(data []byte) ([]byte, error) {
	// Parse v0 config
	var v0 versions.ConfigV0
	if err := yaml.Unmarshal(data, &v0); err != nil {
		return nil, err
	}

	// Create v1 with same data
	v1 := versions.ConfigV1{
		ConfigVersion:         Version1,
		Model:                 v0.Model,
		SummaryModel:          v0.SummaryModel,
		VLModel:               v0.VLModel,
		BaseURL:               v0.BaseURL,
		Provider:              v0.Provider,
		ProviderModels:        v0.ProviderModels,
		ProviderSummaryModels: v0.ProviderSummaryModels,
		ProviderVLModels:      v0.ProviderVLModels,
		Temperature:           v0.Temperature,
		SystemPrompt:          v0.SystemPrompt,
		RequestTimeoutSeconds: v0.RequestTimeoutSeconds,
		ConversationDir:       v0.ConversationDir,
		WorkspaceRoot:         v0.WorkspaceRoot,
		ShellTimeoutSeconds:   v0.ShellTimeoutSeconds,
		ContextProfile:        v0.ContextProfile,
		ZAIBaseURL:            v0.ZAIBaseURL,
		ContextMessagePercent: v0.ContextMessagePercent,
		ContextTotalPercent:   v0.ContextTotalPercent,
		ContextProtectRecent:  v0.ContextProtectRecent,
		MemoryStorePath:       v0.MemoryStorePath,
		HistoryPath:           v0.HistoryPath,
		ThinkingEnabled:       v0.ThinkingEnabled,
		ForceThinking:         v0.ForceThinking,
		CompactionPrompt:      v0.CompactionPrompt,
		OpenRouterFreeMode:    v0.OpenRouterFreeMode,
		AnalyticsEnabled:      v0.AnalyticsEnabled,
	}

	// Apply fixes for common v0 issues

	// 1. Fix thinking defaults - only if field was missing (both false could mean explicit choice)
	// Check if thinking fields were present in original YAML
	var raw map[string]interface{}
	yaml.Unmarshal(data, &raw)

	_, hasThinking := raw["thinking_enabled"]
	_, hasForce := raw["force_thinking"]

	// Only set default if neither field was present
	if !hasThinking && !hasForce && !v1.ThinkingEnabled && !v1.ForceThinking {
		v1.ThinkingEnabled = true
	}

	// 2. Fix context profile - "default" was wrong default, should be "memory"
	if v1.ContextProfile == "" || v1.ContextProfile == "default" {
		v1.ContextProfile = "memory"
	}

	// 3. Fix temperature - 0 means unset, should be 0.7
	if v1.Temperature == 0 {
		v1.Temperature = 0.7
	}

	// 4. Fix context percentages
	if v1.ContextMessagePercent <= 0 {
		v1.ContextMessagePercent = 0.02
	}
	// Fix context_conversation_percent:
	// - If unset (0), set to 0.80
	// - If > 0.80, cap at 0.80 (validation max)
	// - Otherwise keep user's value
	if v1.ContextTotalPercent <= 0 {
		v1.ContextTotalPercent = 0.80
	} else if v1.ContextTotalPercent > 0.80 {
		v1.ContextTotalPercent = 0.80
	}

	// 5. Fix timeout defaults
	if v1.RequestTimeoutSeconds <= 0 {
		v1.RequestTimeoutSeconds = 90
	}
	if v1.ShellTimeoutSeconds <= 0 {
		v1.ShellTimeoutSeconds = 60
	}

	// 6. Fix protect recent
	if v1.ContextProtectRecent <= 0 {
		v1.ContextProtectRecent = 2
	}

	// 7. Fix workspace root
	if v1.WorkspaceRoot == "" {
		v1.WorkspaceRoot = "."
	}

	// 8. Fix provider models maps if empty
	if v1.ProviderModels == nil || len(v1.ProviderModels) == 0 {
		v1.ProviderModels = map[string]string{
			"mock":       "mock-model",
			"openrouter": "deepseek/deepseek-chat-v3-0324",
			"zai":        "glm-4.6",
		}
	}
	if v1.ProviderSummaryModels == nil || len(v1.ProviderSummaryModels) == 0 {
		v1.ProviderSummaryModels = map[string]string{
			"mock":       "mock-summary-model",
			"openrouter": "qwen/qwen3-30b-a3b-instruct-2507",
			"zai":        "glm-4.5-air",
		}
	}
	if v1.ProviderVLModels == nil || len(v1.ProviderVLModels) == 0 {
		v1.ProviderVLModels = map[string]string{
			"mock":       "mock-vl-model",
			"openrouter": "qwen/qwen2.5-vl-32b-instruct",
			"zai":        "glm-4.5v",
		}
	}

	// 9. Fix summary_model if empty and provider is known
	if v1.SummaryModel == "" && v1.Provider != "" {
		if summaryModel, ok := v1.ProviderSummaryModels[v1.Provider]; ok {
			v1.SummaryModel = summaryModel
		}
	}

	// Marshal to YAML
	return yaml.Marshal(&v1)
}
