package contextprofile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cando/internal/state"
	"cando/internal/logging"
	"cando/internal/tooling"
)

type recallMemoryTool struct {
	store *memoryStore
}

func newRecallMemoryTool(store *memoryStore) tooling.Tool {
	return &recallMemoryTool{store: store}
}

func (t *recallMemoryTool) Definition() tooling.ToolDefinition {
	return tooling.ToolDefinition{
		Type: "function",
		Function: tooling.ToolFunction{
			Name:        "recall_memory",
			Description: "Retrieve the full text for a previously summarized memory by ID.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"memory_id": map[string]any{
						"type":        "string",
						"description": "Identifier returned in the memory placeholder (e.g., mem-123).",
					},
				},
				"required": []string{"memory_id"},
			},
		},
	}
}

func (t *recallMemoryTool) Call(ctx context.Context, args map[string]any) (string, error) {
	id, err := argString(args, "memory_id")
	if err != nil || id == "" {
		return "", errors.New("memory_id is required")
	}

	logging.DevLog("memory: recalling %s", id)

	// Access memory and update last access time
	entry, err := t.store.Access(id, func(e *memoryEntry) {
		e.LastAccess = time.Now()
	})
	if err != nil {
		logging.ErrorLog("memory: failed to access %s: %v", id, err)
		return "", err
	}

	// Try to expand in-place if conversation is available in context
	messagesRestored := 0
	expandError := ""
	conv, ok := ConversationFromContext(ctx)
	if ok && conv != nil && len(entry.OriginalMessages) > 0 {
		// Find the placeholder message
		messages := conv.Messages()
		placeholderIdx := -1
		target := fmt.Sprintf("recall_memory(%s", id)
		for i := len(messages) - 1; i >= 0; i-- {
			if strings.Contains(strings.ToLower(messages[i].Content), target) {
				placeholderIdx = i
				break
			}
		}

		// If found, replace with original messages
		if placeholderIdx >= 0 {
			var originalMessages []state.Message
			if err := json.Unmarshal(entry.OriginalMessages, &originalMessages); err != nil {
				// Log unmarshal failure for debugging
				logging.ErrorLog("memory: failed to unmarshal original messages for %s: %v", id, err)
				expandError = fmt.Sprintf("failed to unmarshal original messages: %v", err)
				// Note: we don't return error here to maintain backward compatibility
				// The tool still returns the summary, just can't expand in-place
			} else {
				logging.DevLog("memory: expanded %s with %d original messages", id, len(originalMessages))
				// Replace placeholder with original messages
				before := messages[:placeholderIdx]
				after := messages[placeholderIdx+1:]
				expanded := append(append(before, originalMessages...), after...)
				conv.ReplaceMessages(expanded)
				messagesRestored = len(originalMessages)
			}
		}
	}

	// Return success metadata only (no full content)
	payload := map[string]any{
		"status":            "success",
		"memory_id":         entry.ID,
		"summary":           entry.Summary,
		"messages_restored": messagesRestored,
		"pinned":            entry.Pinned,
		"last_access":       entry.LastAccess.Format(time.RFC3339),
	}
	if expandError != "" {
		payload["expand_error"] = expandError
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type pinMemoryTool struct {
	store   *memoryStore
	maxPins int
}

func newPinMemoryTool(store *memoryStore, maxPins int) tooling.Tool {
	return &pinMemoryTool{store: store, maxPins: maxPins}
}

func (t *pinMemoryTool) Definition() tooling.ToolDefinition {
	return tooling.ToolDefinition{
		Type: "function",
		Function: tooling.ToolFunction{
			Name:        "pin_memory",
			Description: fmt.Sprintf("Protect a memory from compaction. Up to %d memories can be pinned simultaneously.", t.maxPins),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"memory_id": map[string]any{
						"type":        "string",
						"description": "Identifier of the memory to pin or unpin.",
					},
					"pin": map[string]any{
						"type":        "boolean",
						"description": "True to pin (default), false to unpin.",
					},
				},
				"required": []string{"memory_id"},
			},
		},
	}
}

func (t *pinMemoryTool) Call(ctx context.Context, args map[string]any) (string, error) {
	id, err := argString(args, "memory_id")
	if err != nil || id == "" {
		return "", errors.New("memory_id is required")
	}
	pin := argBool(args, "pin", true)
	
	logging.DevLog("memory: %s memory %s", map[bool]string{true: "pinning", false: "unpinning"}[pin], id)
	
	entry, err := t.store.Pin(id, pin, t.maxPins)
	if err != nil {
		logging.ErrorLog("memory: failed to %s %s: %v", map[bool]string{true: "pin", false: "unpin"}[pin], id, err)
		return "", err
	}
	
	logging.UserLog("Memory %s %s successfully (pinned count: %d)", id, map[bool]string{true: "pinned", false: "unpinned"}[pin], t.store.PinnedCount())
	
	payload := map[string]any{
		"memory_id":    entry.ID,
		"pinned":       entry.Pinned,
		"pinned_count": t.store.PinnedCount(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func argString(args map[string]any, key string) (string, error) {
	val, ok := args[key]
	if !ok {
		return "", fmt.Errorf("%s missing", key)
	}
	switch v := val.(type) {
	case string:
		return v, nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func argBool(args map[string]any, key string, defaultVal bool) bool {
	val, ok := args[key]
	if !ok {
		return defaultVal
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return defaultVal
}
