package migrate

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Version constants
const (
	Version0 = 0 // Pre-versioning (main branch)
	Version1 = 1 // First versioned config with fixes

	CurrentVersion = Version1
)

// Migration represents a single migration step
type Migration interface {
	FromVersion() int
	ToVersion() int
	Description() string
	Migrate(data []byte) ([]byte, error)
}

// DetectVersion determines the config version from raw YAML data
func DetectVersion(data []byte) int {
	var probe struct {
		ConfigVersion int `yaml:"config_version"`
	}

	if err := yaml.Unmarshal(data, &probe); err != nil {
		return Version0 // If unmarshal fails, assume v0
	}

	if probe.ConfigVersion == 0 {
		// Could be v0 or v1+ with explicit 0
		// Check for presence of config_version field
		var raw map[string]interface{}
		if err := yaml.Unmarshal(data, &raw); err == nil {
			if _, hasVersion := raw["config_version"]; hasVersion {
				return probe.ConfigVersion
			}
		}
		return Version0 // No version field means v0
	}

	return probe.ConfigVersion
}

// MigrateConfig applies all necessary migrations to bring config to current version
func MigrateConfig(configPath string) error {
	// Read current config
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No config to migrate
		}
		return fmt.Errorf("read config: %w", err)
	}

	// Detect current version
	currentVersion := DetectVersion(data)

	// Already current?
	if currentVersion >= CurrentVersion {
		return nil
	}

	log.Printf("Config migration: v%d → v%d", currentVersion, CurrentVersion)

	// Create backup
	backupDir := filepath.Dir(configPath)
	backupName := fmt.Sprintf("config.yaml.backup.v%d.%s",
		currentVersion, time.Now().Format("20060102-150405"))
	backupPath := filepath.Join(backupDir, backupName)

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("create backup: %w", err)
	}
	log.Printf("Config backed up to: %s", backupPath)

	// Apply migrations in sequence
	migrations := GetMigrationChain(currentVersion, CurrentVersion)

	for _, migration := range migrations {
		log.Printf("Applying migration: %s", migration.Description())
		data, err = migration.Migrate(data)
		if err != nil {
			return fmt.Errorf("migration v%d→v%d failed: %w",
				migration.FromVersion(), migration.ToVersion(), err)
		}
	}

	// Write migrated config
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write migrated config: %w", err)
	}

	log.Printf("Config migration complete")
	return nil
}

// GetMigrationChain returns the sequence of migrations needed
func GetMigrationChain(fromVersion, toVersion int) []Migration {
	var chain []Migration
	current := fromVersion

	for current < toVersion {
		migration := getMigration(current)
		if migration == nil {
			log.Printf("Warning: no migration from v%d", current)
			break
		}
		chain = append(chain, migration)
		current = migration.ToVersion()
	}

	return chain
}

// getMigration returns the migration from a specific version
func getMigration(fromVersion int) Migration {
	switch fromVersion {
	case Version0:
		return &MigrationV0toV1{}
	default:
		return nil
	}
}
