package credentials

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Credentials stores API keys and provider configuration
type Credentials struct {
	DefaultProvider string              `yaml:"default_provider"`
	Providers       map[string]Provider `yaml:"providers"`
}

// Provider stores authentication details for a single provider
type Provider struct {
	APIKey      string `yaml:"api_key"`
	VisionModel string `yaml:"vision_model,omitempty"`
}

// Manager handles credential storage and retrieval
type Manager struct {
	path string
}

// NewManager creates a new credential manager
// Checks CANDO_CREDENTIALS_PATH environment variable first.
// If not set, defaults to ~/.cando/credentials.yaml
func NewManager() (*Manager, error) {
	credPath := os.Getenv("CANDO_CREDENTIALS_PATH")
	if credPath == "" {
		configDir := getConfigDir()
		credPath = filepath.Join(configDir, "credentials.yaml")
	}

	return &Manager{path: credPath}, nil
}

func getConfigDir() string {
	if configDir := os.Getenv("CANDO_CONFIG_DIR"); configDir != "" {
		return configDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".cando"
	}
	return filepath.Join(home, ".cando")
}

// Load reads credentials from disk
func (m *Manager) Load() (*Credentials, error) {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty credentials if file doesn't exist
			return &Credentials{
				Providers: make(map[string]Provider),
			}, nil
		}
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	var creds Credentials
	if err := yaml.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}

	if creds.Providers == nil {
		creds.Providers = make(map[string]Provider)
	}

	return &creds, nil
}

// Save writes credentials to disk
func (m *Manager) Save(creds *Credentials) error {
	// Ensure directory exists
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}

	data, err := yaml.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	// Write with restricted permissions (user-only read/write)
	if err := os.WriteFile(m.path, data, 0600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}

	return nil
}

// Exists checks if credentials file exists
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.path)
	return err == nil
}

// Path returns the credentials file path
func (m *Manager) Path() string {
	return m.path
}

// IsConfigured checks if a provider is configured
func (c *Credentials) IsConfigured(provider string) bool {
	if c.Providers == nil {
		return false
	}
	p, exists := c.Providers[provider]
	return exists && p.APIKey != ""
}

// GetAPIKey returns the API key for a provider
func (c *Credentials) GetAPIKey(provider string) string {
	if c.Providers == nil {
		return ""
	}
	return c.Providers[provider].APIKey
}

// SetProvider sets the API key for a provider
func (c *Credentials) SetProvider(name, apiKey string) {
	if c.Providers == nil {
		c.Providers = make(map[string]Provider)
	}
	c.Providers[name] = Provider{APIKey: apiKey}
}

// RemoveProvider removes a provider
func (c *Credentials) RemoveProvider(name string) {
	if c.Providers != nil {
		delete(c.Providers, name)
	}
}

// HasAnyProvider checks if any provider is configured
func (c *Credentials) HasAnyProvider() bool {
	if c.Providers == nil {
		return false
	}
	for _, p := range c.Providers {
		if p.APIKey != "" {
			return true
		}
	}
	return false
}

// ListProviders returns all configured provider names
func (c *Credentials) ListProviders() []string {
	if c.Providers == nil {
		return nil
	}
	names := make([]string, 0, len(c.Providers))
	for name, p := range c.Providers {
		if p.APIKey != "" {
			names = append(names, name)
		}
	}
	return names
}

// GetVisionModel returns the vision model for a provider
func (c *Credentials) GetVisionModel(provider string) string {
	if c.Providers == nil {
		return ""
	}
	return c.Providers[provider].VisionModel
}
