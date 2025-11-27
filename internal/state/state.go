package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	// ErrUnknownState is returned when operations reference an undefined key.
	ErrUnknownState = errors.New("unknown state")

	fileExtension = ".json"
	keySanitizer  = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
)

// Message mirrors the OpenAI/OpenRouter chat schema so that stored history can be
// reused verbatim in requests.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	Thinking   string     `json:"thinking,omitempty"`
}

// ToolCall represents a function call request emitted by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall is embedded inside ToolCall for OpenAI-compatible schemas.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Conversation is a named, mutable list of chat messages with persistence metadata.
type Conversation struct {
	key         string
	messages    []Message
	storagePath string
	createdAt   time.Time
	updatedAt   time.Time
}

// Key returns the identifier assigned to the conversation.
func (c *Conversation) Key() string {
	return c.key
}

// StoragePath returns the file path where this conversation is persisted.
func (c *Conversation) StoragePath() string {
	return c.storagePath
}

// Messages exposes the underlying history for serialization.
func (c *Conversation) Messages() []Message {
	out := make([]Message, len(c.messages))
	copy(out, c.messages)
	return out
}

// Append adds a new chat message to the history.
func (c *Conversation) Append(msg Message) {
	c.messages = append(c.messages, msg)
	c.touch()
}

// Clear removes all non-system history and reinstates the system prompt when given.
func (c *Conversation) Clear(systemPrompt string) {
	c.messages = c.messages[:0]
	if systemPrompt != "" {
		c.messages = append(c.messages, Message{Role: "system", Content: systemPrompt})
	}
	c.touch()
}

// ReplaceMessages swaps the current conversation history with the provided slice.
func (c *Conversation) ReplaceMessages(messages []Message) {
	c.messages = make([]Message, len(messages))
	copy(c.messages, messages)
	c.touch()
}

// CreatedAt returns when the conversation was first persisted.
func (c *Conversation) CreatedAt() time.Time {
	return c.createdAt
}

// UpdatedAt returns when the conversation last changed.
func (c *Conversation) UpdatedAt() time.Time {
	return c.updatedAt
}

func (c *Conversation) touch() {
	now := time.Now()
	if c.createdAt.IsZero() {
		c.createdAt = now
	}
	c.updatedAt = now
}

// Manager orchestrates multiple named conversations.
type Manager struct {
	mu           sync.RWMutex
	states       map[string]*Conversation
	currentKey   string
	systemPrompt string
	root         string
	logger       *log.Logger
}

// NewManager sets up the container for managing multiple contexts backed by disk persistence.
func NewManager(systemPrompt, root string, logger *log.Logger) (*Manager, error) {
	if root == "" {
		root = "conversations"
	}
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create conversation dir: %w", err)
	}
	mgr := &Manager{
		states:       make(map[string]*Conversation),
		systemPrompt: systemPrompt,
		root:         root,
		logger:       logger,
	}
	if err := mgr.loadExisting(); err != nil {
		return nil, err
	}
	return mgr, nil
}

// EnsureState fetches or creates a conversation for the provided key.
func (m *Manager) EnsureState(key string) (*Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if key == "" {
		key = m.generateUniqueSessionNameLocked()
	}
	if conv, ok := m.states[key]; ok {
		m.currentKey = key
		return conv, nil
	}
	conv := newConversation(key, m.systemPrompt)
	if err := m.assignPathLocked(conv); err != nil {
		return nil, err
	}
	if err := m.persistConversationLocked(conv); err != nil {
		return nil, err
	}
	m.states[key] = conv
	m.currentKey = key
	return conv, nil
}

// NewState explicitly creates a fresh conversation and errors if the key exists.
func (m *Manager) NewState(key string) (*Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.states[key]; exists {
		return nil, fmt.Errorf("state %s already exists", key)
	}
	conv := newConversation(key, m.systemPrompt)
	if err := m.assignPathLocked(conv); err != nil {
		return nil, err
	}
	if err := m.persistConversationLocked(conv); err != nil {
		return nil, err
	}
	m.states[key] = conv
	m.currentKey = key
	return conv, nil
}

// Use switches to an existing conversation.
func (m *Manager) Use(key string) (*Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	conv, ok := m.states[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownState, key)
	}
	m.currentKey = key
	return conv, nil
}

// Delete removes a stored conversation from memory and disk.
func (m *Manager) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	conv, ok := m.states[key]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownState, key)
	}
	if conv.storagePath != "" {
		if err := os.Remove(conv.storagePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete state %s: %w", key, err)
		}
	}
	delete(m.states, key)
	if m.currentKey == key {
		m.currentKey = ""
	}
	return nil
}

// Current exposes the active conversation, creating a default one if needed.
func (m *Manager) Current() *Conversation {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ensureCurrentLocked()
}

// CurrentKey reveals which conversation is active.
func (m *Manager) CurrentKey() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentKey
}

// ListKeys returns the known conversation identifiers.
func (m *Manager) ListKeys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.states))
	for k := range m.states {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Summary captures metadata about a stored conversation without exposing message content.
type Summary struct {
	Key          string    `json:"key"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
}

// Summaries returns lightweight details for each known conversation, sorted by last update desc.
func (m *Manager) Summaries() []Summary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	summaries := make([]Summary, 0, len(m.states))
	for key, conv := range m.states {
		if conv == nil {
			continue
		}
		summaries = append(summaries, Summary{
			Key:          key,
			CreatedAt:    conv.CreatedAt(),
			UpdatedAt:    conv.UpdatedAt(),
			MessageCount: len(conv.messages),
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt)
	})
	return summaries
}

// ClearCurrent wipes the active conversation history.
func (m *Manager) ClearCurrent() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	conv := m.ensureCurrentLocked()
	conv.Clear(m.systemPrompt)
	return m.persistConversationLocked(conv)
}

// SetSystemPrompt updates the default system prompt used for new conversations.
func (m *Manager) SetSystemPrompt(prompt string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.systemPrompt = prompt
}

// Save writes the provided conversation to disk.
func (m *Manager) Save(conv *Conversation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conv == nil {
		return fmt.Errorf("conversation is nil")
	}
	if _, ok := m.states[conv.key]; !ok {
		return fmt.Errorf("%w: %s", ErrUnknownState, conv.key)
	}
	return m.persistConversationLocked(conv)
}

func (m *Manager) ensureCurrentLocked() *Conversation {
	if m.currentKey == "" {
		m.currentKey = m.generateUniqueSessionNameLocked()
	}
	if conv, ok := m.states[m.currentKey]; ok {
		return conv
	}
	conv := newConversation(m.currentKey, m.systemPrompt)
	if err := m.assignPathLocked(conv); err != nil {
		m.logger.Printf("assign storage path failed: %v", err)
	} else if err := m.persistConversationLocked(conv); err != nil {
		m.logger.Printf("persist conversation failed: %v", err)
	}
	m.states[m.currentKey] = conv
	return conv
}

func (m *Manager) loadExisting() error {
	entries, err := os.ReadDir(m.root)
	if err != nil {
		return fmt.Errorf("read conversation root: %w", err)
	}
	loaded := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dayDir := filepath.Join(m.root, entry.Name())
		files, err := os.ReadDir(dayDir)
		if err != nil {
			m.logger.Printf("skip %s: %v", dayDir, err)
			continue
		}
		for _, fileEntry := range files {
			if fileEntry.IsDir() || filepath.Ext(fileEntry.Name()) != fileExtension {
				continue
			}
			path := filepath.Join(dayDir, fileEntry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				m.logger.Printf("read %s failed: %v", path, err)
				continue
			}
			var persisted persistedConversation
			if err := json.Unmarshal(data, &persisted); err != nil {
				m.logger.Printf("parse %s failed: %v", path, err)
				continue
			}
			key := persisted.Key
			if key == "" {
				key = strings.TrimSuffix(fileEntry.Name(), fileExtension)
			}
			conv := &Conversation{
				key:         key,
				messages:    persisted.Messages,
				storagePath: path,
				createdAt:   persisted.CreatedAt,
				updatedAt:   persisted.UpdatedAt,
			}
			if conv.createdAt.IsZero() {
				if info, statErr := os.Stat(path); statErr == nil {
					conv.createdAt = info.ModTime()
				} else {
					conv.createdAt = time.Now()
				}
			}
			if conv.updatedAt.IsZero() {
				conv.updatedAt = conv.createdAt
			}
			if existing, exists := m.states[conv.key]; exists {
				if existing.updatedAt.After(conv.updatedAt) {
					continue
				}
			}
			m.states[conv.key] = conv
			loaded++
		}
	}
	if loaded > 0 {
		m.logger.Printf("loaded %d stored conversations", loaded)

		// Set current key to most recently updated session
		var mostRecent *Conversation
		for _, conv := range m.states {
			if mostRecent == nil || conv.updatedAt.After(mostRecent.updatedAt) {
				mostRecent = conv
			}
		}
		if mostRecent != nil {
			m.currentKey = mostRecent.key
		}
	}
	return nil
}

func (m *Manager) assignPathLocked(conv *Conversation) error {
	if conv.storagePath != "" {
		return nil
	}
	folder := filepath.Join(m.root, conv.createdAt.Format("2006-01-02"))
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return fmt.Errorf("create folder %s: %w", folder, err)
	}
	sanitized := sanitizeKey(conv.key)
	conv.storagePath = filepath.Join(folder, sanitized+fileExtension)
	return nil
}

func (m *Manager) persistConversationLocked(conv *Conversation) error {
	if conv.storagePath == "" {
		if err := m.assignPathLocked(conv); err != nil {
			return err
		}
	}
	payload := persistedConversation{
		Key:       conv.key,
		Messages:  conv.messages,
		CreatedAt: conv.createdAt,
		UpdatedAt: conv.updatedAt,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal conversation: %w", err)
	}
	tmp := conv.storagePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp conversation: %w", err)
	}
	if err := os.Rename(tmp, conv.storagePath); err != nil {
		return fmt.Errorf("replace conversation: %w", err)
	}
	return nil
}

func sanitizeKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "conversation"
	}
	sanitized := keySanitizer.ReplaceAllString(trimmed, "_")
	sanitized = strings.Trim(sanitized, "_-")
	if sanitized == "" {
		sanitized = "conversation"
	}
	return sanitized
}

// generateUniqueSessionNameLocked creates a unique sequential session name (chat-1, chat-2, etc.).
// Caller must hold m.mu lock.
func (m *Manager) generateUniqueSessionNameLocked() string {
	maxNum := 0
	for key := range m.states {
		var num int
		if _, err := fmt.Sscanf(key, "chat-%d", &num); err == nil {
			if num > maxNum {
				maxNum = num
			}
		}
	}
	return fmt.Sprintf("chat-%d", maxNum+1)
}

func newConversation(key, systemPrompt string) *Conversation {
	now := time.Now()
	conv := &Conversation{key: key, createdAt: now, updatedAt: now}
	if systemPrompt != "" {
		conv.messages = append(conv.messages, Message{Role: "system", Content: systemPrompt})
	}
	return conv
}

// persistedConversation mirrors the JSON schema stored on disk.
type persistedConversation struct {
	Key       string    `json:"key"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
