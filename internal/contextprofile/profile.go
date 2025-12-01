package contextprofile

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"cando/internal/config"
	"cando/internal/llm"
	"cando/internal/state"
	"cando/internal/tooling"
)

// Prepared encapsulates the conversation snapshot returned by a profile before an LLM call.
type Prepared struct {
	Messages []state.Message
	Mutated  bool
}

// Profile defines the hooks used to customize conversation context.
type Profile interface {
	Prepare(ctx context.Context, conv *state.Conversation) (Prepared, error)
	AfterResponse(ctx context.Context, conv *state.Conversation) (bool, error)
	Tools() []tooling.Tool
	SetToolDefinitions(defs []tooling.ToolDefinition)
}

type ConfigReloadable interface {
	ReloadConfig(cfg config.Config) error
}

// CompactionEvent represents a single compaction operation's statistics.
type CompactionEvent struct {
	Timestamp          time.Time `json:"timestamp"`
	CharsBefore        int       `json:"chars_before"`
	CharsAfter         int       `json:"chars_after"`
	MessagesCompacted  int       `json:"messages_compacted"`
	MessagesConsidered int       `json:"messages_considered"`
	DurationMs         int64     `json:"duration_ms"`
}

// CompactionEventEmitter is an optional interface for profiles that support compaction event emission.
type CompactionEventEmitter interface {
	SetCompactionCallback(callback func(eventType string, data any) error)
	GetCompactionHistory() []CompactionEvent
}

// FactsExtractor is called before compaction to extract project knowledge from the conversation.
// Implementation should handle loading existing facts, calling LLM, and saving updated facts.
type FactsExtractor interface {
	ExtractFacts(ctx context.Context, messages []state.Message) error
}

// FactsExtractorSetter is an optional interface for profiles that support facts extraction.
type FactsExtractorSetter interface {
	SetFactsExtractor(fe FactsExtractor)
}

// Dependencies bundles the resources profiles may require.
type Dependencies struct {
	Client   llm.Client
	Logger   *log.Logger
	Config   config.Config
	Provider string // Active provider (e.g., "zai", "openrouter")
	Model    string // Active model name
}

// New selects the requested profile by name.
func New(name string, deps Dependencies) (Profile, error) {
	switch strings.ToLower(name) {
	case "", "default":
		return &noopProfile{}, nil
	case "memory":
		profile, err := newMemoryProfile(deps)
		if err != nil {
			return nil, err
		}
		return profile, nil
	default:
		return nil, fmt.Errorf("unknown context profile %s", name)
	}
}

type noopProfile struct{}

func (noopProfile) Prepare(_ context.Context, conv *state.Conversation) (Prepared, error) {
	return Prepared{Messages: conv.Messages()}, nil
}

func (noopProfile) AfterResponse(_ context.Context, _ *state.Conversation) (bool, error) {
	return false, nil
}

func (noopProfile) Tools() []tooling.Tool { return nil }

func (noopProfile) SetToolDefinitions([]tooling.ToolDefinition) {
	// No-op for default profile
}
