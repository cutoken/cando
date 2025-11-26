package config

import (
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
				ContextMessagePercent:  0.02,
				ContextTotalPercent:    0.50,
				ContextProtectRecent:   5,
				Temperature:            0.7,
				RequestTimeoutSeconds:  90,
				ShellTimeoutSeconds:    60,
				MemoryStorePath:        "/tmp/memory.db",
				HistoryPath:            "/tmp/.history",
				SummaryModel:           "glm-4.5-air",
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
