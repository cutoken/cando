package config

import "testing"

func TestGetModelContextLength(t *testing.T) {
	tests := []struct {
		provider string
		model    string
		expected int
	}{
		// Z.AI models
		{"zai", "glm-4.6", 200000},
		{"zai", "glm-4.5", 128000},
		{"zai", "glm-4.5-air", 128000},
		{"ZAI", "glm-4.6", 200000}, // Test case insensitive provider

		// OpenRouter models (sample)
		{"openrouter", "anthropic/claude-3.5-sonnet", 200000},

		// Unknown model - should return default
		{"unknown-provider", "unknown-model", 65536},
		{"zai", "non-existent-model", 65536},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.model, func(t *testing.T) {
			result := GetModelContextLength(tt.provider, tt.model)
			if result != tt.expected {
				t.Errorf("GetModelContextLength(%q, %q) = %d, want %d", tt.provider, tt.model, result, tt.expected)
			}
		})
	}
}

func TestGetAllModelContexts(t *testing.T) {
	contexts := GetAllModelContexts()

	// Verify we have entries
	if len(contexts) == 0 {
		t.Error("GetAllModelContexts() returned empty map")
	}

	// Verify Z.AI models are present
	if _, ok := contexts["zai/glm-4.6"]; !ok {
		t.Error("Expected zai/glm-4.6 to be in contexts")
	}

	// Verify OpenRouter models are present (check for any openrouter/ key)
	hasOpenRouter := false
	for key := range contexts {
		if len(key) > 11 && key[:11] == "openrouter/" {
			hasOpenRouter = true
			break
		}
	}
	if !hasOpenRouter {
		t.Error("Expected at least one openrouter model in contexts")
	}
}
