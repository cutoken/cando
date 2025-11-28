package agent

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cando/internal/config"
)

// Workspace represents a folder selected by the user
type Workspace struct {
	Path  string    `json:"path"`  // Absolute path to folder
	Slug  string    `json:"slug"`  // Generated slug for storage (name-hash)
	Name  string    `json:"name"`  // Display name (folder basename)
	Added time.Time `json:"added"` // When workspace was added
}

// WorkspaceManager handles workspace list persistence and operations
type WorkspaceManager struct {
	mu         sync.RWMutex
	workspaces []Workspace
	recent     []Workspace
	current    string // Current workspace path
	filePath   string // Path to workspaces.json
}

type workspaceFile struct {
	Workspaces []Workspace `json:"workspaces"`
	Recent     []Workspace `json:"recent,omitempty"`
	Current    string      `json:"current"`
}

// NewWorkspaceManager creates a workspace manager with persistence
func NewWorkspaceManager() (*WorkspaceManager, error) {
	candoDir := config.GetConfigDir()
	if err := os.MkdirAll(candoDir, 0o755); err != nil {
		return nil, fmt.Errorf("create .cando dir: %w", err)
	}

	filePath := filepath.Join(candoDir, "workspaces.json")
	mgr := &WorkspaceManager{
		workspaces: []Workspace{},
		filePath:   filePath,
	}

	// Load existing workspaces if file exists
	if err := mgr.load(); err != nil {
		// If file doesn't exist, that's fine - we'll create it on first save
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("load workspaces: %w", err)
		}
	}

	return mgr, nil
}

// Add adds a new workspace to the list
func (m *WorkspaceManager) Add(path string) (*Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Normalize path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	// Check if already added
	for _, w := range m.workspaces {
		if w.Path == absPath {
			m.removeFromRecentLocked(absPath)
			return &w, nil // Already exists, return it
		}
	}

	// Create workspace
	ws := Workspace{
		Path:  absPath,
		Slug:  generateSlug(absPath),
		Name:  filepath.Base(absPath),
		Added: time.Now(),
	}

	m.workspaces = append(m.workspaces, ws)
	m.removeFromRecentLocked(absPath)

	// Set as current if it's the first one
	if m.current == "" {
		m.current = absPath
	}

	if err := m.saveLocked(); err != nil {
		return nil, fmt.Errorf("save workspaces: %w", err)
	}

	return &ws, nil
}

// Remove removes a workspace from the list
func (m *WorkspaceManager) Remove(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Find and remove
	found := false
	var removed Workspace
	newList := make([]Workspace, 0, len(m.workspaces))
	for _, w := range m.workspaces {
		if w.Path == absPath {
			found = true
			removed = w
			continue
		}
		newList = append(newList, w)
	}

	if !found {
		return fmt.Errorf("workspace not found: %s", absPath)
	}

	// Prevent removing the last workspace
	m.workspaces = newList
	if found {
		m.addToRecentLocked(removed)
	}

	// If we removed the current workspace, switch to the first one (if any)
	if m.current == absPath {
		if len(m.workspaces) > 0 {
			m.current = m.workspaces[0].Path
		} else {
			m.current = ""
		}
	}

	return m.saveLocked()
}

// SetCurrent sets the current workspace
func (m *WorkspaceManager) SetCurrent(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Verify it exists in our list
	found := false
	for _, w := range m.workspaces {
		if w.Path == absPath {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("workspace not in list: %s", absPath)
	}

	m.current = absPath
	m.removeFromRecentLocked(absPath)
	return m.saveLocked()
}

// Current returns the current workspace
func (m *WorkspaceManager) Current() *Workspace {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, w := range m.workspaces {
		if w.Path == m.current {
			return &w
		}
	}

	// Fallback to first if current is invalid
	if len(m.workspaces) > 0 {
		return &m.workspaces[0]
	}

	return nil
}

// List returns all workspaces
func (m *WorkspaceManager) List() []Workspace {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Workspace, len(m.workspaces))
	copy(result, m.workspaces)
	return result
}

// GetByPath returns a workspace by path
func (m *WorkspaceManager) GetByPath(path string) *Workspace {
	m.mu.RLock()
	defer m.mu.RUnlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil
	}

	for _, w := range m.workspaces {
		if w.Path == absPath {
			return &w
		}
	}

	return nil
}

// load reads workspaces from disk
func (m *WorkspaceManager) load() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	var file workspaceFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse workspaces.json: %w", err)
	}

	m.workspaces = file.Workspaces
	m.recent = file.Recent
	m.current = file.Current

	return nil
}

// saveLocked writes workspaces to disk (caller must hold lock)
func (m *WorkspaceManager) saveLocked() error {
	file := workspaceFile{
		Workspaces: m.workspaces,
		Recent:     m.recent,
		Current:    m.current,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal workspaces: %w", err)
	}

	if err := os.WriteFile(m.filePath, data, 0o644); err != nil {
		return fmt.Errorf("write workspaces.json: %w", err)
	}

	return nil
}

// Recent returns the list of recently closed workspaces.
func (m *WorkspaceManager) Recent() []Workspace {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Workspace, len(m.recent))
	copy(result, m.recent)
	return result
}

func (m *WorkspaceManager) addToRecentLocked(ws Workspace) {
	for _, existing := range m.workspaces {
		if existing.Path == ws.Path {
			return
		}
	}
	for _, r := range m.recent {
		if r.Path == ws.Path {
			return
		}
	}
	m.recent = append([]Workspace{ws}, m.recent...)
	if len(m.recent) > 5 {
		m.recent = m.recent[:5]
	}
}

func (m *WorkspaceManager) removeFromRecentLocked(path string) {
	if len(m.recent) == 0 {
		return
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return
	}
	filtered := m.recent[:0]
	for _, r := range m.recent {
		if r.Path != absPath {
			filtered = append(filtered, r)
		}
	}
	m.recent = filtered
}

// generateSlug creates a unique slug from workspace path
func generateSlug(path string) string {
	clean := filepath.Clean(path)
	base := sanitizeSlug(filepath.Base(clean))
	if base == "" {
		base = "workspace"
	}
	sum := sha1.Sum([]byte(clean))
	hash := hex.EncodeToString(sum[:8])
	return fmt.Sprintf("%s-%s", base, hash)
}

// sanitizeSlug removes non-alphanumeric characters from a string
func sanitizeSlug(name string) string {
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		} else if r == ' ' {
			result.WriteRune('-')
		}
	}
	return strings.ToLower(result.String())
}

// ProjectStorageRoot returns the storage directory for a workspace
// Returns: ~/.cando/projects/<slug>/
func ProjectStorageRoot(workspace string) (string, error) {
	slug := generateSlug(workspace)
	return filepath.Join(config.GetConfigDir(), "projects", slug), nil
}
