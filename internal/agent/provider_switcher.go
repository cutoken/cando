package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"cando/internal/llm"
)

// ProviderOption describes a selectable provider/model combination exposed to the UI.
type ProviderOption struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Model  string `json:"model"`
	Source string `json:"source,omitempty"`
}

// ProviderRegistration wires a client implementation with the metadata necessary to expose it.
type ProviderRegistration struct {
	Option ProviderOption
	Client llm.Client
}

// ProviderSwitcher exposes the ability to switch between providers/models at runtime.
type ProviderSwitcher interface {
	ActiveProvider() ProviderOption
	ProviderOptions() []ProviderOption
	SetActiveProvider(key string) error
}

type multiProviderClient struct {
	mu        sync.RWMutex
	activeKey string
	entries   map[string]providerEntry
}

type providerEntry struct {
	option ProviderOption
	client llm.Client
}

// NewMultiProviderClient builds a Chat client capable of switching between the registered providers.
func NewMultiProviderClient(defaultKey string, regs []ProviderRegistration) (llm.Client, error) {
	if len(regs) == 0 {
		return nil, fmt.Errorf("no provider registrations supplied")
	}
	entries := make(map[string]providerEntry, len(regs))
	for _, reg := range regs {
		key := strings.TrimSpace(reg.Option.Key)
		if key == "" {
			return nil, fmt.Errorf("provider registration missing key")
		}
		if reg.Client == nil {
			return nil, fmt.Errorf("provider %s missing client", key)
		}
		entries[key] = providerEntry{
			option: reg.Option,
			client: reg.Client,
		}
	}
	active := defaultKey
	if _, ok := entries[active]; !ok {
		for key := range entries {
			active = key
			break
		}
	}
	return &multiProviderClient{
		activeKey: active,
		entries:   entries,
	}, nil
}

func (m *multiProviderClient) Chat(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	entry, err := m.activeEntry()
	if err != nil {
		return llm.ChatResponse{}, err
	}
	if entry.option.Model != "" {
		req.Model = entry.option.Model
	}
	return entry.client.Chat(ctx, req)
}

func (m *multiProviderClient) activeEntry() (providerEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.entries[m.activeKey]
	if !ok {
		return providerEntry{}, fmt.Errorf("active provider %q unavailable", m.activeKey)
	}
	return entry, nil
}

func (m *multiProviderClient) ActiveProvider() ProviderOption {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.entries[m.activeKey]
	if !ok {
		return ProviderOption{}
	}
	return entry.option
}

func (m *multiProviderClient) ProviderOptions() []ProviderOption {
	m.mu.RLock()
	defer m.mu.RUnlock()
	opts := make([]ProviderOption, 0, len(m.entries))
	for _, entry := range m.entries {
		opts = append(opts, entry.option)
	}
	sort.Slice(opts, func(i, j int) bool {
		return opts[i].Label < opts[j].Label
	})
	return opts
}

func (m *multiProviderClient) SetActiveProvider(key string) error {
	key = strings.TrimSpace(key)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.entries[key]; !ok {
		return fmt.Errorf("provider %q not available", key)
	}
	m.activeKey = key
	return nil
}
