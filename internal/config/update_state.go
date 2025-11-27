package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// UpdateState stores auto-update check state
type UpdateState struct {
	LastCheckTime  time.Time `json:"lastCheckTime"`
	LatestVersion  string    `json:"latestVersion"`
	DismissedUntil time.Time `json:"dismissedUntil"`
}

const updateStateFile = "update-state.json"

// LoadUpdateState loads update state from disk
func LoadUpdateState() (*UpdateState, error) {
	path := filepath.Join(GetConfigDir(), updateStateFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &UpdateState{}, nil
		}
		return nil, err
	}
	var state UpdateState
	if err := json.Unmarshal(data, &state); err != nil {
		return &UpdateState{}, nil // Return empty on parse error
	}
	return &state, nil
}

// Save persists update state to disk
func (s *UpdateState) Save() error {
	path := filepath.Join(GetConfigDir(), updateStateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// NeedsCheck returns true if we should check for updates (weekly)
func (s *UpdateState) NeedsCheck() bool {
	return time.Since(s.LastCheckTime) > 7*24*time.Hour
}

// IsDismissed returns true if user dismissed the update prompt
func (s *UpdateState) IsDismissed() bool {
	return time.Now().Before(s.DismissedUntil)
}

// Dismiss sets dismissal for 7 days
func (s *UpdateState) Dismiss() {
	s.DismissedUntil = time.Now().Add(7 * 24 * time.Hour)
}

// RecordCheck updates last check time and latest version
func (s *UpdateState) RecordCheck(version string) {
	s.LastCheckTime = time.Now()
	s.LatestVersion = version
}
