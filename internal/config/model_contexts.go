package config

import (
	_ "embed"
	"encoding/json"
	"log"
	"strings"
	"sync"
)

//go:embed model-contexts.json
var modelContextsJSON []byte

var (
	modelContexts     map[string]int
	modelContextsOnce sync.Once
)

// loadModelContexts lazily loads the embedded model context lengths JSON
func loadModelContexts() {
	modelContextsOnce.Do(func() {
		modelContexts = make(map[string]int)
		if err := json.Unmarshal(modelContextsJSON, &modelContexts); err != nil {
			log.Printf("Warning: failed to parse model contexts: %v", err)
		}
	})
}

// GetModelContextLength returns the maximum context length for a given provider and model.
// Returns 65536 as a safe default if the model is not found.
// Provider examples: "openrouter", "zai"
// Model examples: "anthropic/claude-3.5-sonnet", "glm-4.6"
func GetModelContextLength(provider, model string) int {
	loadModelContexts()

	// Normalize provider to lowercase
	provider = strings.ToLower(strings.TrimSpace(provider))
	model = strings.ToLower(strings.TrimSpace(model))

	// Build lookup key: "provider/model"
	key := provider + "/" + model

	if contextLength, ok := modelContexts[key]; ok && contextLength > 0 {
		return contextLength
	}

	// Return safe default for unknown models
	return 65536
}

// GetAllModelContexts returns the entire map of model contexts.
// Useful for debugging or UI display.
func GetAllModelContexts() map[string]int {
	loadModelContexts()
	// Return a copy to prevent external modification
	result := make(map[string]int, len(modelContexts))
	for k, v := range modelContexts {
		result[k] = v
	}
	return result
}
