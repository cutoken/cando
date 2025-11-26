package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		modifyFunc  func(*Config)
		expectError bool
		errorString string
	}{
		{
			name: "valid config passes",
			modifyFunc: func(c *Config) {
				c.ContextMessagePercent = 0.02
				c.ContextTotalPercent = 0.50
				c.Temperature = 0.7
				c.RequestTimeoutSeconds = 90
				c.ShellTimeoutSeconds = 60
			},
			expectError: false,
		},
		{
			name: "ContextMessagePercent exceeds ContextTotalPercent fails",
			modifyFunc: func(c *Config) {
				c.ContextMessagePercent = 0.60
				c.ContextTotalPercent = 0.50
			},
			expectError: true,
			errorString: "context_message_percent",
		},
		{
			name: "negative temperature fails",
			modifyFunc: func(c *Config) {
				c.Temperature = -0.5
			},
			expectError: true,
			errorString: "temperature must be between",
		},
		{
			name: "temperature > 2.0 fails",
			modifyFunc: func(c *Config) {
				c.Temperature = 3.0
			},
			expectError: true,
			errorString: "temperature must be between",
		},
		{
			name: "request timeout > 600 fails",
			modifyFunc: func(c *Config) {
				c.RequestTimeoutSeconds = 9999
			},
			expectError: true,
			errorString: "request_timeout_seconds cannot exceed",
		},
		{
			name: "shell timeout > 600 fails",
			modifyFunc: func(c *Config) {
				c.ShellTimeoutSeconds = 9999
			},
			expectError: true,
			errorString: "shell_timeout_seconds cannot exceed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create valid base config
			cfg := Config{
				ContextMessagePercent: 0.02,
				ContextTotalPercent:   0.50,
				ContextProtectRecent:  5,
				Temperature:           0.7,
				RequestTimeoutSeconds: 90,
				ShellTimeoutSeconds:   60,
				MemoryStorePath:       "/tmp/memory.db",
				HistoryPath:           "/tmp/.history",
				SummaryModel:          "glm-4.5-air",
			}

			// Apply test-specific modifications
			tt.modifyFunc(&cfg)

			// Run validation
			err := cfg.validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorString) {
					t.Errorf("Expected error containing %q, got %q", tt.errorString, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestDefaultCompactionPromptConsistency(t *testing.T) {
	// Verify that EnsureDefaultConfig uses the same prompt as the constant
	cfg := Config{}
	cfg.CompactionPrompt = DefaultCompactionPrompt

	if cfg.CompactionPrompt != DefaultCompactionPrompt {
		t.Errorf("CompactionPrompt mismatch: got %q, want %q", cfg.CompactionPrompt, DefaultCompactionPrompt)
	}

	// Verify the constant value
	if DefaultCompactionPrompt != "Summarize the following text in 20 words or fewer. Return only the summary." {
		t.Errorf("DefaultCompactionPrompt changed unexpectedly: %q", DefaultCompactionPrompt)
	}
}

func TestNewUserDefaultsByProvider(t *testing.T) {
	tests := []struct {
		name            string
		provider        string
		expectedModel   string
		expectedSummary string
		expectedVL      string
	}{
		{
			name:            "ZAI provider gets correct defaults",
			provider:        "zai",
			expectedModel:   DefaultZAIModel,
			expectedSummary: DefaultZAISummaryModel,
			expectedVL:      DefaultZAIVLModel,
		},
		{
			name:            "OpenRouter provider gets correct defaults",
			provider:        "openrouter",
			expectedModel:   DefaultOpenRouterModel,
			expectedSummary: DefaultOpenRouterSummaryModel,
			expectedVL:      DefaultOpenRouterVLModel,
		},
		{
			name:            "Mock provider gets OpenRouter defaults (not handled in EnsureDefaultConfig)",
			provider:        "mock",
			expectedModel:   DefaultOpenRouterModel, // Falls through to default case
			expectedSummary: "", // Not set for unknown providers
			expectedVL:      "", // Not set for unknown providers
		},
		{
			name:            "Unknown provider gets OpenRouter defaults",
			provider:        "unknown",
			expectedModel:   DefaultOpenRouterModel,
			expectedSummary: "", // Not set in EnsureDefaultConfig for unknown providers
			expectedVL:      "", // Not set in EnsureDefaultConfig for unknown providers
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for test config
			tempDir := t.TempDir()
			configDir := filepath.Join(tempDir, ".cando")
			configPath := filepath.Join(configDir, "config.yaml")

			// Override the home directory for this test
			originalHome := os.Getenv("HOME")
			t.Cleanup(func() {
				os.Setenv("HOME", originalHome)
			})
			os.Setenv("HOME", tempDir)

			// Ensure config doesn't exist initially
			if _, err := os.Stat(configPath); err == nil {
				t.Fatalf("Config file should not exist before test: %s", configPath)
			}

			// Run EnsureDefaultConfig
			err := EnsureDefaultConfig(tt.provider)
			if err != nil {
				t.Fatalf("EnsureDefaultConfig failed: %v", err)
			}

			// Verify config was created
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				t.Fatalf("Config file was not created: %s", configPath)
			}

			// Load and verify the config
			cfg, err := LoadUserConfig()
			if err != nil {
				t.Fatalf("LoadUserConfig failed: %v", err)
			}

			// Check main model
			if cfg.Model != tt.expectedModel {
				t.Errorf("Expected Model %q, got %q", tt.expectedModel, cfg.Model)
			}

			// Check provider-specific model
			if len(cfg.ProviderModels) > 0 && cfg.ProviderModels[tt.provider] != tt.expectedModel {
				t.Errorf("Expected ProviderModels[%s] %q, got %q", tt.provider, tt.expectedModel, cfg.ProviderModels[tt.provider])
			}

			// Check summary model if expected
			if tt.expectedSummary != "" {
				if cfg.SummaryModel != tt.expectedSummary {
					t.Errorf("Expected SummaryModel %q, got %q", tt.expectedSummary, cfg.SummaryModel)
				}
			}

			// Check VL model if expected
			if tt.expectedVL != "" {
				if cfg.VLModel != tt.expectedVL {
					t.Errorf("Expected VLModel %q, got %q", tt.expectedVL, cfg.VLModel)
				}
				// Also check provider-specific VL model
				if len(cfg.ProviderVLModels) > 0 && cfg.ProviderVLModels[tt.provider] != tt.expectedVL {
					t.Errorf("Expected ProviderVLModels[%s] %q, got %q", tt.provider, tt.expectedVL, cfg.ProviderVLModels[tt.provider])
				}
			}
		})
	}
}

func TestModelForProviderFallbacks(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		setupConfig  func(*Config)
		expectedModel string
	}{
		{
			name:     "ZAI gets provider-specific model when configured",
			provider: "zai",
			setupConfig: func(c *Config) {
				c.ProviderModels = map[string]string{"zai": "custom-zai-model"}
			},
			expectedModel: "custom-zai-model",
		},
		{
			name:     "ZAI falls back to default when not configured",
			provider: "zai",
			setupConfig: func(c *Config) {
				c.ProviderModels = map[string]string{} // Empty
			},
			expectedModel: DefaultZAIModel,
		},
		{
			name:     "OpenRouter gets provider-specific model when configured",
			provider: "openrouter",
			setupConfig: func(c *Config) {
				c.ProviderModels = map[string]string{"openrouter": "custom-openrouter-model"}
			},
			expectedModel: "custom-openrouter-model",
		},
		{
			name:     "OpenRouter falls back to default when not configured",
			provider: "openrouter",
			setupConfig: func(c *Config) {
				c.ProviderModels = map[string]string{} // Empty
			},
			expectedModel: DefaultOpenRouterModel,
		},
		{
			name:     "Mock gets default model",
			provider: "mock",
			setupConfig: func(c *Config) {
				c.ProviderModels = map[string]string{} // Empty
			},
			expectedModel: DefaultMockModel,
		},
		{
			name:     "Unknown provider falls back to generic model",
			provider: "unknown",
			setupConfig: func(c *Config) {
				c.ProviderModels = map[string]string{} // Empty
				c.Model = "generic-model"
			},
			expectedModel: "generic-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			tt.setupConfig(cfg)

			result := cfg.ModelFor(tt.provider)
			if result != tt.expectedModel {
				t.Errorf("Expected ModelFor(%s) %q, got %q", tt.provider, tt.expectedModel, result)
			}
		})
	}
}

func TestSummaryModelForProviderFallbacks(t *testing.T) {
	tests := []struct {
		name              string
		provider          string
		setupConfig       func(*Config)
		expectedSummaryModel string
	}{
		{
			name:     "ZAI gets provider-specific summary model when configured",
			provider: "zai",
			setupConfig: func(c *Config) {
				c.ProviderSummaryModels = map[string]string{"zai": "custom-zai-summary"}
			},
			expectedSummaryModel: "custom-zai-summary",
		},
		{
			name:     "ZAI falls back to default summary when not configured",
			provider: "zai",
			setupConfig: func(c *Config) {
				c.ProviderSummaryModels = map[string]string{} // Empty
			},
			expectedSummaryModel: DefaultZAISummaryModel,
		},
		{
			name:     "OpenRouter gets provider-specific summary model when configured",
			provider: "openrouter",
			setupConfig: func(c *Config) {
				c.ProviderSummaryModels = map[string]string{"openrouter": "custom-openrouter-summary"}
			},
			expectedSummaryModel: "custom-openrouter-summary",
		},
		{
			name:     "OpenRouter falls back to default summary when not configured",
			provider: "openrouter",
			setupConfig: func(c *Config) {
				c.ProviderSummaryModels = map[string]string{} // Empty
			},
			expectedSummaryModel: DefaultOpenRouterSummaryModel,
		},
		{
			name:     "Mock gets default summary model",
			provider: "mock",
			setupConfig: func(c *Config) {
				c.ProviderSummaryModels = map[string]string{} // Empty
			},
			expectedSummaryModel: DefaultMockSummaryModel,
		},
		{
			name:     "Unknown provider falls back to generic summary model",
			provider: "unknown",
			setupConfig: func(c *Config) {
				c.ProviderSummaryModels = map[string]string{} // Empty
				c.SummaryModel = "generic-summary-model"
			},
			expectedSummaryModel: "generic-summary-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			tt.setupConfig(cfg)

			result := cfg.SummaryModelFor(tt.provider)
			if result != tt.expectedSummaryModel {
				t.Errorf("Expected SummaryModelFor(%s) %q, got %q", tt.provider, tt.expectedSummaryModel, result)
			}
		})
	}
}

func TestVLModelForProviderFallbacks(t *testing.T) {
	tests := []struct {
		name            string
		provider        string
		setupConfig     func(*Config)
		expectedVLModel string
	}{
		{
			name:     "ZAI gets provider-specific VL model when configured",
			provider: "zai",
			setupConfig: func(c *Config) {
				c.ProviderVLModels = map[string]string{"zai": "custom-zai-vl"}
			},
			expectedVLModel: "custom-zai-vl",
		},
		{
			name:     "OpenRouter gets provider-specific VL model when configured",
			provider: "openrouter",
			setupConfig: func(c *Config) {
				c.ProviderVLModels = map[string]string{"openrouter": "custom-openrouter-vl"}
			},
			expectedVLModel: "custom-openrouter-vl",
		},
		{
			name:     "ZAI falls back to default VL when not configured",
			provider: "zai",
			setupConfig: func(c *Config) {
				c.ProviderVLModels = map[string]string{} // Empty
			},
			expectedVLModel: DefaultZAIVLModel,
		},
		{
			name:     "OpenRouter falls back to default VL when not configured",
			provider: "openrouter",
			setupConfig: func(c *Config) {
				c.ProviderVLModels = map[string]string{} // Empty
			},
			expectedVLModel: DefaultOpenRouterVLModel,
		},
		{
			name:     "Mock gets default VL model",
			provider: "mock",
			setupConfig: func(c *Config) {
				c.ProviderVLModels = map[string]string{}
			},
			expectedVLModel: DefaultMockVLModel,
		},
		{
			name:     "Unknown provider falls back to generic VL model",
			provider: "unknown",
			setupConfig: func(c *Config) {
				c.ProviderVLModels = map[string]string{}
				c.VLModel = "generic-vl-model"
			},
			expectedVLModel: "generic-vl-model",
		},
		{
			name:     "Unknown provider falls back to OpenRouter VL when no generic model",
			provider: "unknown",
			setupConfig: func(c *Config) {
				c.ProviderVLModels = map[string]string{}
				// No generic VL model set
			},
			expectedVLModel: DefaultOpenRouterVLModel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			tt.setupConfig(cfg)

			result := cfg.VLModelFor(tt.provider)
			if result != tt.expectedVLModel {
				t.Errorf("Expected VLModelFor(%s) %q, got %q", tt.provider, tt.expectedVLModel, result)
			}
		})
	}
}
