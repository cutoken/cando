package contextprofile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"cando/internal/config"
	"cando/internal/llm"
	"cando/internal/state"
	"cando/internal/tooling"
)

const (
	memoryCooldown             = 2 * time.Minute
	memoryMaxPins              = 5
	memoryPlaceholderIndicator = "[compacted thread:"
)

// convCtxKey is used to pass conversation state through context to tools
type convCtxKey struct{}

// WithConversation adds a conversation to the context for tool access
func WithConversation(ctx context.Context, conv *state.Conversation) context.Context {
	return context.WithValue(ctx, convCtxKey{}, conv)
}

// ConversationFromContext extracts the conversation from context if present
func ConversationFromContext(ctx context.Context) (*state.Conversation, bool) {
	conv, ok := ctx.Value(convCtxKey{}).(*state.Conversation)
	return conv, ok
}

type ProtectedSetter interface {
	SetProtectedRecent(int)
}

type CompactionForcer interface {
	ForceCompaction()
}

type MemoryInspector interface {
	MemorySummary(limit int) (MemorySummary, error)
}

type MemorySummary struct {
	Total   int
	Pinned  int
	Entries []MemorySummaryEntry
}

type MemorySummaryEntry struct {
	ID         string
	Summary    string
	Pinned     bool
	LastAccess time.Time
}

type memoryProfile struct {
	client                llm.Client
	logger                *log.Logger
	model                 string
	summaryModel          string
	store                 *memoryStore
	cfg                   config.Config
	provider              string
	threshold             int
	conversationThreshold int
	protectedRecent       int
	cooldown              time.Duration
	maxPins               int
	randSrc               *rand.Rand
	mu                    sync.RWMutex
	skipCompaction        bool
	forceCompaction       bool
	summaryPrompt         string
	compactionHistory     []CompactionEvent
	compactionCallback    func(eventType string, data any) error
	toolDefinitions       []tooling.ToolDefinition
	toolDefsMu            sync.RWMutex
	factsExtractor        FactsExtractor
}

func (p *memoryProfile) SetProtectedRecent(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if n < 0 {
		n = 0
	}
	p.protectedRecent = n
}

func (p *memoryProfile) currentProtected() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.protectedRecent
}

func (p *memoryProfile) deferCompaction() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.skipCompaction = true
}

func (p *memoryProfile) shouldSkipCompaction() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.skipCompaction
}

func (p *memoryProfile) clearSkipCompaction() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.skipCompaction = false
}

func (p *memoryProfile) ForceCompaction() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.forceCompaction = true
}

func (p *memoryProfile) shouldForceCompaction() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.forceCompaction
}

func (p *memoryProfile) clearForceCompaction() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.forceCompaction = false
}

// SetFactsExtractor sets the facts extractor to be called before compaction.
func (p *memoryProfile) SetFactsExtractor(fe FactsExtractor) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.factsExtractor = fe
}

func (p *memoryProfile) getFactsExtractor() FactsExtractor {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.factsExtractor
}

func newMemoryProfile(deps Dependencies) (*memoryProfile, error) {
	if deps.Client == nil {
		return nil, errors.New("memory profile requires llm client")
	}
	logger := deps.Logger
	if logger == nil {
		logger = log.Default()
	}
	model := deps.Model
	if model == "" {
		model = deps.Config.Model
		if model == "" {
			model = "qwen/qwen2-80b-instruct"
		}
	}
	provider := deps.Provider
	if provider == "" {
		provider = deps.Config.Provider
	}

	// Get provider-specific summary model
	summaryModel := deps.Config.SummaryModelFor(provider)

	// Calculate absolute thresholds from percentages
	messageLimit := deps.Config.CalculateMessageThreshold(provider, model)
	totalLimit := deps.Config.CalculateConversationThreshold(provider, model)
	protected := deps.Config.ContextProtectRecent

	store, err := newMemoryStore(deps.Config.MemoryStorePath)
	if err != nil {
		return nil, err
	}

	// Load compaction history from database
	history, err := store.LoadCompactionEvents()
	if err != nil {
		logger.Printf("Warning: failed to load compaction history: %v", err)
		history = []CompactionEvent{} // Continue with empty history
	}

	return &memoryProfile{
		client:                deps.Client,
		logger:                logger,
		model:                 model,
		summaryModel:          summaryModel,
		store:                 store,
		cfg:                   deps.Config,
		provider:              provider,
		threshold:             messageLimit,
		conversationThreshold: totalLimit,
		protectedRecent:       protected,
		cooldown:              memoryCooldown,
		maxPins:               memoryMaxPins,
		randSrc:               rand.New(rand.NewSource(time.Now().UnixNano())),
		summaryPrompt:         deps.Config.CompactionPrompt,
		compactionHistory:     history,
	}, nil
}

func (p *memoryProfile) Prepare(ctx context.Context, conv *state.Conversation) (Prepared, error) {
	messages := conv.Messages()
	mutated := false

	if p.shouldSkipCompaction() {
		return Prepared{Messages: messages}, nil
	}

	total := p.totalActualSize(messages)
	forced := p.shouldForceCompaction()
	if forced {
		p.clearForceCompaction()
	}

	if total > p.conversationThreshold || forced {
		// Extract project facts before compaction (while we have full context)
		if fe := p.getFactsExtractor(); fe != nil {
			if err := fe.ExtractFacts(ctx, messages); err != nil {
				// Log but don't block compaction on facts extraction failure
				p.logger.Printf("facts extraction failed: %v", err)
			}
		}

		stats, err := p.compactOverflow(ctx, messages, total, forced)
		if err != nil {
			return Prepared{}, err
		}
		if stats != nil {
			if stats.compacted > 0 {
				// Remove empty message shells left by compaction
				messages = removeEmptyMessages(messages)
				mutated = true
				if forced {
					p.logger.Printf("FORCED context compaction: %d -> %d chars across %d messages (considered=%d)", stats.before, stats.after, stats.compacted, stats.considered)
					fmt.Printf("(forced compaction reduced context from %d to %d chars; %d messages summarized)\n", stats.before, stats.after, stats.compacted)
				} else {
					p.logger.Printf("context compaction: %d -> %d chars across %d messages (considered=%d)", stats.before, stats.after, stats.compacted, stats.considered)
					fmt.Printf("(compaction reduced context from %d to %d chars; %d messages summarized)\n", stats.before, stats.after, stats.compacted)
				}
			} else {
				if forced {
					p.logger.Printf("forced context compaction attempted but no eligible messages (total=%d limit=%d considered=%d)", stats.before, p.conversationThreshold, stats.considered)
				} else {
					p.logger.Printf("context compaction attempted but no eligible messages (total=%d limit=%d considered=%d)", stats.before, p.conversationThreshold, stats.considered)
				}
			}
		}
	}

	if mutated {
		conv.ReplaceMessages(messages)
		return Prepared{Messages: conv.Messages(), Mutated: true}, nil
	}
	return Prepared{Messages: messages}, nil
}

func (p *memoryProfile) AfterResponse(_ context.Context, _ *state.Conversation) (bool, error) {
	p.clearSkipCompaction()
	return false, nil
}

func (p *memoryProfile) Tools() []tooling.Tool {
	return []tooling.Tool{
		newRecallMemoryTool(p.store),
		newPinMemoryTool(p.store, p.maxPins),
	}
}

func (p *memoryProfile) SetToolDefinitions(defs []tooling.ToolDefinition) {
	p.toolDefsMu.Lock()
	defer p.toolDefsMu.Unlock()
	p.toolDefinitions = defs
}

func (p *memoryProfile) getToolDefinitions() []tooling.ToolDefinition {
	p.toolDefsMu.RLock()
	defer p.toolDefsMu.RUnlock()
	return p.toolDefinitions
}

type compactionStats struct {
	before     int
	after      int
	compacted  int
	considered int
}

// turnBoundary represents a range of messages that form a single assistant turn
type turnBoundary struct {
	startIdx int // inclusive
	endIdx   int // inclusive
}

// identifyTurns finds turn boundaries in the message list.
// A turn consists of consecutive assistant/tool messages ending when:
// - Assistant message has no tool_calls, OR
// - We hit a user/system message
func identifyTurns(messages []state.Message) []turnBoundary {
	var turns []turnBoundary
	var currentTurnStart = -1

	for i := 0; i < len(messages); i++ {
		msg := &messages[i]
		role := strings.ToLower(msg.Role)

		// User and system messages are never part of a turn
		if role == "user" || role == "system" {
			// End current turn if one is active
			if currentTurnStart >= 0 {
				turns = append(turns, turnBoundary{
					startIdx: currentTurnStart,
					endIdx:   i - 1,
				})
				currentTurnStart = -1
			}
			continue
		}

		// Assistant or tool messages are part of a turn
		if role == "assistant" || role == "tool" {
			// Start a new turn if not already in one
			if currentTurnStart < 0 {
				currentTurnStart = i
			}

			// Check if this assistant message ends the turn (no tool calls)
			if role == "assistant" && len(msg.ToolCalls) == 0 {
				turns = append(turns, turnBoundary{
					startIdx: currentTurnStart,
					endIdx:   i,
				})
				currentTurnStart = -1
			}
		}
	}

	// Close any open turn at the end
	if currentTurnStart >= 0 {
		turns = append(turns, turnBoundary{
			startIdx: currentTurnStart,
			endIdx:   len(messages) - 1,
		})
	}

	return turns
}

// compactTurn compacts an entire turn (multiple messages) into a single placeholder.
// Returns: delta (chars saved), compacted (whether compaction happened), error
func (p *memoryProfile) compactTurn(ctx context.Context, messages []state.Message, turn turnBoundary) (int, bool, error) {
	if turn.startIdx < 0 || turn.endIdx >= len(messages) || turn.startIdx > turn.endIdx {
		p.logger.Printf("compactTurn: SKIP turn[%d:%d] - invalid indices (len=%d)", turn.startIdx, turn.endIdx, len(messages))
		return 0, false, nil
	}

	// Extract original messages for this turn
	turnMessages := make([]state.Message, turn.endIdx-turn.startIdx+1)
	copy(turnMessages, messages[turn.startIdx:turn.endIdx+1])

	// Pre-check: scan all messages in turn for placeholders
	hasPlaceholder := false
	allPlaceholders := true
	for i := turn.startIdx; i <= turn.endIdx; i++ {
		if isPlaceholder(messages[i].Content) {
			hasPlaceholder = true
		} else if messages[i].Content != "" || messages[i].Thinking != "" {
			allPlaceholders = false
		}
	}

	// If all messages are already placeholders, skip (already compacted)
	if allPlaceholders && hasPlaceholder {
		p.logger.Printf("compactTurn: SKIP turn[%d:%d] - already compacted (all placeholders)", turn.startIdx, turn.endIdx)
		return 0, false, nil
	}

	// If some (but not all) messages are placeholders, inconsistent state - skip with warning
	if hasPlaceholder {
		p.logger.Printf("compactTurn: SKIP turn[%d:%d] - mixed state (some placeholders, some content), avoiding corruption",
			turn.startIdx, turn.endIdx)
		return 0, false, nil
	}

	// Aggregate all content from messages in this turn for summarization
	var contentBuilder strings.Builder
	originalSize := 0
	hasContent := false

	for i := turn.startIdx; i <= turn.endIdx; i++ {
		msg := &messages[i]

		if msg.Content != "" {
			hasContent = true
			contentBuilder.WriteString(fmt.Sprintf("[%s", msg.Role))
			if msg.Name != "" {
				contentBuilder.WriteString(fmt.Sprintf(" %s", msg.Name))
			}
			contentBuilder.WriteString("]: ")
			contentBuilder.WriteString(msg.Content)
			contentBuilder.WriteString("\n\n")
			originalSize += len(msg.Content)
		}

		// Include thinking if present
		if msg.Thinking != "" {
			contentBuilder.WriteString(fmt.Sprintf("[thinking]: %s\n\n", msg.Thinking))
			originalSize += len(msg.Thinking)
		}
	}

	// Skip if turn has no actual content to summarize
	if !hasContent {
		p.logger.Printf("compactTurn: SKIP turn[%d:%d] - no content to summarize", turn.startIdx, turn.endIdx)
		return 0, false, nil
	}

	// Create memory entry with both content summary and full message chain
	aggregatedContent := contentBuilder.String()
	entry, err := p.createMemory(ctx, aggregatedContent, turnMessages)
	if err != nil {
		return 0, false, err
	}

	// Replace all messages in turn with a single placeholder message
	// Keep the first message and clear the rest
	messages[turn.startIdx].Content = entry.Placeholder
	messages[turn.startIdx].Thinking = ""
	messages[turn.startIdx].ToolCalls = nil

	// Mark other messages in the turn for removal by clearing their content
	for i := turn.startIdx + 1; i <= turn.endIdx; i++ {
		messages[i].Content = ""
		messages[i].Thinking = ""
		messages[i].ToolCalls = nil
	}

	delta := originalSize - len(entry.Placeholder)
	if delta < 0 {
		delta = 0
	}

	return delta, true, nil
}

func (p *memoryProfile) compactOverflow(ctx context.Context, messages []state.Message, total int, forced bool) (*compactionStats, error) {
	startTime := time.Now()
	stats := &compactionStats{
		before: total,
	}

	// Emit compaction start event
	p.emitCompactionEvent("compaction_start", map[string]any{
		"chars_before": total,
	})

	current := total
	protect := p.currentProtected()
	if protect < 0 {
		protect = 0
	}
	protectedStartIdx := len(messages) - protect
	if protectedStartIdx < 0 {
		protectedStartIdx = 0
	}

	// Log compaction parameters
	p.logger.Printf("compaction: totalMessages=%d, protect=%d, protectedStartIdx=%d", len(messages), protect, protectedStartIdx)

	// Identify turns in the conversation
	turns := identifyTurns(messages)

	// Log turns identified
	p.logger.Printf("compaction: identified %d turns", len(turns))
	for i, turn := range turns {
		p.logger.Printf("  turn[%d]: startIdx=%d, endIdx=%d", i, turn.startIdx, turn.endIdx)
	}

	// Filter turns to only those that end before the protected range
	var compactableTurns []turnBoundary
	for _, turn := range turns {
		if turn.endIdx < protectedStartIdx {
			compactableTurns = append(compactableTurns, turn)
		}
	}

	// Log compactable turns
	p.logger.Printf("compaction: %d turns are compactable (endIdx <= %d)", len(compactableTurns), protectedStartIdx)

	stats.considered = len(compactableTurns)

	// Compact turns from oldest to newest
	for i, turn := range compactableTurns {
		// When forcing, skip threshold check and compact all eligible turns
		if !forced && current <= p.conversationThreshold {
			p.logger.Printf("compaction: stopped at turn %d/%d (current=%d <= threshold=%d)", i, len(compactableTurns), current, p.conversationThreshold)
			break
		}
		p.logger.Printf("compaction: attempting turn %d/%d (startIdx=%d, endIdx=%d, current=%d)", i+1, len(compactableTurns), turn.startIdx, turn.endIdx, current)
		_, changed, err := p.compactTurn(ctx, messages, turn)
		if err != nil {
			p.logger.Printf("compaction: turn %d FAILED: %v", i+1, err)
			continue
		}
		if changed {
			// Recalculate actual size from JSON after compaction
			// Note: Empty shells will be removed by caller (Prepare)
			newSize := p.totalActualSize(messages)
			p.logger.Printf("compaction: turn %d COMPACTED: %d -> %d chars (saved %d)", i+1, current, newSize, current-newSize)
			current = newSize
			stats.compacted++
		} else {
			p.logger.Printf("compaction: turn %d SKIPPED: no change", i+1)
		}
	}

	p.logger.Printf("compaction: finished - compacted %d/%d turns, %d -> %d chars", stats.compacted, len(compactableTurns), stats.before, current)
	stats.after = current
	duration := time.Since(startTime)

	// Create and store compaction event
	event := CompactionEvent{
		Timestamp:          startTime,
		CharsBefore:        stats.before,
		CharsAfter:         stats.after,
		MessagesCompacted:  stats.compacted,
		MessagesConsidered: stats.considered,
		DurationMs:         duration.Milliseconds(),
	}
	p.addCompactionEvent(event)

	// Emit compaction complete event
	p.emitCompactionEvent("compaction_complete", event)

	return stats, nil
}

func (p *memoryProfile) compactMessage(ctx context.Context, msg *state.Message) (int, bool, error) {
	if msg == nil || msg.Content == "" {
		return 0, false, nil
	}
	role := strings.ToLower(msg.Role)
	if role == "system" {
		return 0, false, nil
	}
	if isPlaceholder(msg.Content) {
		return 0, false, nil
	}
	if len(msg.Content) <= p.threshold {
		return 0, false, nil
	}
	// For single message compaction, store the message itself
	originalMessages := []state.Message{*msg}
	entry, err := p.createMemory(ctx, msg.Content, originalMessages)
	if err != nil {
		return 0, false, err
	}
	original := len(msg.Content)
	msg.Content = entry.Placeholder
	delta := original - len(msg.Content)
	if delta < 0 {
		delta = 0
	}
	return delta, true, nil
}

func (p *memoryProfile) createMemory(ctx context.Context, content string, originalMessages []state.Message) (*memoryEntry, error) {
	summary, err := p.summarize(ctx, content)
	if err != nil {
		return nil, err
	}
	id := p.generateID()
	placeholder := fmt.Sprintf("[COMPACTED THREAD: %s]\nI've summarized this thread segment. Summary: %s\nI can recall with recall_memory(%s) if details are needed.", id, summary, id)

	// Marshal original messages to JSON for storage
	var originalMessagesJSON []byte
	if len(originalMessages) > 0 {
		originalMessagesJSON, err = json.Marshal(originalMessages)
		if err != nil {
			return nil, fmt.Errorf("marshal original messages: %w", err)
		}
	}

	entry := &memoryEntry{
		ID:               id,
		Content:          content,
		Summary:          summary,
		Placeholder:      placeholder,
		OriginalMessages: originalMessagesJSON,
		CreatedAt:        time.Now(),
		LastAccess:       time.Now(),
	}
	if err := p.store.Put(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (p *memoryProfile) summarize(ctx context.Context, content string) (string, error) {
	resp, err := p.client.Chat(ctx, llm.ChatRequest{
		Model: p.summaryModel,
		Messages: []state.Message{
			{Role: "system", Content: p.summaryPrompt},
			{Role: "user", Content: content},
		},
		Temperature: 0.1,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("no summary returned")
	}
	summary := strings.TrimSpace(resp.Choices[0].Message.Content)
	if summary == "" {
		return "", errors.New("empty summary")
	}
	if wordCount(summary) > 20 {
		summary = truncateWords(summary, 20)
	}
	return summary, nil
}

func (p *memoryProfile) generateID() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return fmt.Sprintf("mem-%d-%04x", time.Now().UnixNano(), p.randSrc.Intn(0xffff))
}

func isPlaceholder(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, memoryPlaceholderIndicator)
}

func totalContentLength(messages []state.Message) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content)
	}
	return total
}

func (p *memoryProfile) totalActualSize(messages []state.Message) int {
	// Marshal messages to JSON
	msgData, err := json.Marshal(messages)
	if err != nil {
		// Fallback to content-only measurement
		return totalContentLength(messages)
	}

	// Get tool definitions
	toolDefs := p.getToolDefinitions()
	if len(toolDefs) == 0 {
		// No tools set yet, use estimate
		return len(msgData) + 10000
	}

	// Marshal tools to JSON
	toolData, err := json.Marshal(toolDefs)
	if err != nil {
		// Fallback to estimate
		return len(msgData) + 10000
	}

	return len(msgData) + len(toolData)
}

func (p *memoryProfile) MemorySummary(limit int) (MemorySummary, error) {
	if limit <= 0 {
		limit = 5
	}
	total, pinned, entries, err := p.store.Stats(limit)
	if err != nil {
		return MemorySummary{}, err
	}
	view := make([]MemorySummaryEntry, 0, len(entries))
	for _, entry := range entries {
		view = append(view, MemorySummaryEntry{
			ID:         entry.ID,
			Summary:    entry.Summary,
			Pinned:     entry.Pinned,
			LastAccess: entry.LastAccess,
		})
	}
	return MemorySummary{
		Total:   total,
		Pinned:  pinned,
		Entries: view,
	}, nil
}

func (p *memoryProfile) ReloadConfig(cfg config.Config) error {
	if strings.TrimSpace(cfg.MemoryStorePath) != "" && cfg.MemoryStorePath != p.store.Path() {
		return fmt.Errorf("changing memory_store_path requires restart")
	}
	if cfg.ContextMessagePercent <= 0 || cfg.ContextTotalPercent <= 0 {
		return fmt.Errorf("invalid compaction thresholds")
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	// Recalculate absolute thresholds from percentages
	p.cfg = cfg
	p.threshold = cfg.CalculateMessageThreshold(p.provider, p.model)
	p.conversationThreshold = cfg.CalculateConversationThreshold(p.provider, p.model)

	if cfg.ContextProtectRecent >= 0 {
		p.protectedRecent = cfg.ContextProtectRecent
	}
	if strings.TrimSpace(cfg.CompactionPrompt) != "" {
		p.summaryPrompt = cfg.CompactionPrompt
	}
	// Update summary model using provider-specific value if available
	summaryModel := cfg.SummaryModelFor(p.provider)
	if summaryModel != "" {
		p.summaryModel = summaryModel
	}
	p.skipCompaction = false
	return nil
}

// UpdateProviderModel updates the active provider and model, recalculating compaction thresholds
func (p *memoryProfile) UpdateProviderModel(provider, model string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.provider = provider
	p.model = model

	// Recalculate thresholds with new provider/model
	p.threshold = p.cfg.CalculateMessageThreshold(provider, model)
	p.conversationThreshold = p.cfg.CalculateConversationThreshold(provider, model)

	// Update summary model for new provider
	summaryModel := p.cfg.SummaryModelFor(provider)
	if summaryModel != "" {
		p.summaryModel = summaryModel
	}

	p.logger.Printf("Updated compaction thresholds for %s/%s: message=%d, conversation=%d, summary_model=%s",
		provider, model, p.threshold, p.conversationThreshold, p.summaryModel)
}

// SetCompactionCallback implements CompactionEventEmitter.
func (p *memoryProfile) SetCompactionCallback(callback func(eventType string, data any) error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.compactionCallback = callback
}

// GetCompactionHistory implements CompactionEventEmitter.
func (p *memoryProfile) GetCompactionHistory() []CompactionEvent {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// Return a copy to prevent external modification
	history := make([]CompactionEvent, len(p.compactionHistory))
	copy(history, p.compactionHistory)
	return history
}

func (p *memoryProfile) addCompactionEvent(event CompactionEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.compactionHistory = append(p.compactionHistory, event)
	// Keep only last 50 events
	if len(p.compactionHistory) > 50 {
		p.compactionHistory = p.compactionHistory[len(p.compactionHistory)-50:]
	}

	// Persist to database
	if err := p.store.SaveCompactionEvent(event); err != nil {
		p.logger.Printf("Failed to save compaction event to database: %v", err)
	}
}

func (p *memoryProfile) emitCompactionEvent(eventType string, data any) {
	p.mu.RLock()
	callback := p.compactionCallback
	p.mu.RUnlock()
	if callback != nil {
		if err := callback(eventType, data); err != nil {
			p.logger.Printf("compaction event emission failed: %v", err)
		}
	}
}

func wordCount(s string) int {
	fields := strings.Fields(s)
	return len(fields)
}

func truncateWords(s string, limit int) string {
	fields := strings.Fields(s)
	if len(fields) <= limit {
		return s
	}
	return strings.Join(fields[:limit], " ")
}

// removeEmptyMessages filters out empty message shells left by compaction
func removeEmptyMessages(messages []state.Message) []state.Message {
	filtered := make([]state.Message, 0, len(messages))
	for _, msg := range messages {
		// Keep message if it has any content, thinking, or tool calls
		if msg.Content != "" || msg.Thinking != "" || len(msg.ToolCalls) > 0 {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}
