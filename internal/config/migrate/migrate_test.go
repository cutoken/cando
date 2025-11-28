package migrate

import (
	"cando/internal/config/versions"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDetectVersion(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected int
	}{
		{
			name: "v0 without version field",
			yaml: `model: gpt-4
temperature: 0.7`,
			expected: Version0,
		},
		{
			name: "v1 with version field",
			yaml: `config_version: 1
model: gpt-4`,
			expected: Version1,
		},
		{
			name:     "empty config",
			yaml:     ``,
			expected: Version0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := DetectVersion([]byte(tt.yaml))
			if version != tt.expected {
				t.Errorf("DetectVersion() = %d, want %d", version, tt.expected)
			}
		})
	}
}

func TestMigrationV0toV1(t *testing.T) {
	tests := []struct {
		name     string
		testFile string
		validate func(t *testing.T, v1 *versions.ConfigV1)
	}{
		{
			name:     "minimal v0",
			testFile: "testdata/v0_minimal.yaml",
			validate: func(t *testing.T, v1 *versions.ConfigV1) {
				if v1.ConfigVersion != Version1 {
					t.Errorf("ConfigVersion = %d, want %d", v1.ConfigVersion, Version1)
				}
				if !v1.ThinkingEnabled {
					t.Error("ThinkingEnabled should be true for new configs")
				}
				if v1.ContextProfile != "memory" {
					t.Errorf("ContextProfile = %s, want 'memory'", v1.ContextProfile)
				}
			},
		},
		{
			name:     "full v0 with wrong defaults",
			testFile: "testdata/v0_full.yaml",
			validate: func(t *testing.T, v1 *versions.ConfigV1) {
				if v1.ContextProfile != "memory" {
					t.Errorf("ContextProfile = %s, want 'memory' (fixed from 'default')", v1.ContextProfile)
				}
				if !v1.ThinkingEnabled {
					t.Error("ThinkingEnabled should be true when not set")
				}
				if v1.ContextMessagePercent != 0.02 {
					t.Errorf("ContextMessagePercent = %f, want 0.02", v1.ContextMessagePercent)
				}
				if v1.ContextTotalPercent != 0.80 {
					t.Errorf("ContextTotalPercent = %f, want 0.80", v1.ContextTotalPercent)
				}
				if v1.RequestTimeoutSeconds != 90 {
					t.Errorf("RequestTimeoutSeconds = %d, want 90", v1.RequestTimeoutSeconds)
				}
			},
		},
		{
			name:     "v0 with explicit thinking disabled",
			testFile: "testdata/v0_with_thinking.yaml",
			validate: func(t *testing.T, v1 *versions.ConfigV1) {
				if v1.ThinkingEnabled {
					t.Error("ThinkingEnabled should remain false when explicitly set")
				}
				if v1.ContextProfile != "memory" {
					t.Errorf("ContextProfile = %s, want 'memory' (fixed from empty)", v1.ContextProfile)
				}
			},
		},
		{
			name:     "v0 with high context percent",
			testFile: "testdata/v0_high_context.yaml",
			validate: func(t *testing.T, v1 *versions.ConfigV1) {
				if v1.ContextTotalPercent != 0.80 {
					t.Errorf("ContextTotalPercent = %f, want 0.80 (capped from 0.9)", v1.ContextTotalPercent)
				}
				if !v1.ThinkingEnabled {
					t.Error("ThinkingEnabled should be true")
				}
				if v1.ContextProfile != "memory" {
					t.Errorf("ContextProfile = %s, want 'memory'", v1.ContextProfile)
				}
			},
		},
	}

	migration := &MigrationV0toV1{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read test file
			data, err := os.ReadFile(tt.testFile)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			// Run migration
			result, err := migration.Migrate(data)
			if err != nil {
				t.Fatalf("Migration failed: %v", err)
			}

			// Parse result
			var v1 versions.ConfigV1
			if err := yaml.Unmarshal(result, &v1); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}

			// Validate
			tt.validate(t, &v1)
		})
	}
}

func TestMigrationChain(t *testing.T) {
	chain := GetMigrationChain(Version0, Version1)

	if len(chain) != 1 {
		t.Errorf("Expected 1 migration in chain, got %d", len(chain))
	}

	if chain[0].FromVersion() != Version0 || chain[0].ToVersion() != Version1 {
		t.Error("Wrong migration in chain")
	}
}
