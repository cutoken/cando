package contextprofile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"cando/internal/config"
	"cando/internal/llm"
	"cando/internal/state"
	"cando/internal/tooling"
)

// TestCompactionWithRealSession tests compaction on a real 98-message session
func TestCompactionWithRealSession(t *testing.T) {
	// Load real session
	testDataPath := filepath.Join("testdata", "jungle-journey.json")
	data, err := os.ReadFile(testDataPath)
	if err != nil {
		t.Fatalf("Failed to load test session: %v", err)
	}

	var sessionData struct {
		Messages []state.Message `json:"messages"`
	}
	if err := json.Unmarshal(data, &sessionData); err != nil {
		t.Fatalf("Failed to parse test session: %v", err)
	}

	originalMessages := sessionData.Messages
	originalCount := len(originalMessages)
	originalJSON, _ := json.Marshal(originalMessages)
	originalSize := len(originalJSON)

	t.Logf("Loaded session: %d messages, %d bytes", originalCount, originalSize)

	// Setup test environment
	ctx := context.Background()
	cfg := config.Config{
		MemoryStorePath:       filepath.Join(t.TempDir(), "test_memory.db"),
		ContextMessagePercent: 0.02,  // 2%
		ContextTotalPercent:   0.25,  // 25%
		ContextProtectRecent:  2,
	}

	// Create mock client
	mockClient := &mockLLMClient{
		summaries: make(map[string]string),
	}

	// Create memory profile
	profile, err := newMemoryProfile(Dependencies{
		Client:   mockClient,
		Config:   cfg,
		Provider: "test",
		Model:    "test-model",
	})
	if err != nil {
		t.Fatalf("Failed to create memory profile: %v", err)
	}
	defer profile.store.Close()

	// Create conversation
	conv := newTestConversation(originalMessages)

	// Run compaction
	prepared, err := profile.Prepare(ctx, conv)
	if err != nil {
		t.Fatalf("Compaction failed: %v", err)
	}

	compactedMessages := prepared.Messages
	compactedCount := len(compactedMessages)
	compactedJSON, _ := json.Marshal(compactedMessages)
	compactedSize := len(compactedJSON)

	t.Logf("After compaction: %d messages, %d bytes", compactedCount, compactedSize)
	t.Logf("Reduction: %d messages removed, %d bytes saved (%.1f%%)",
		originalCount-compactedCount,
		originalSize-compactedSize,
		float64(originalSize-compactedSize)/float64(originalSize)*100)

	// TEST 1: Verify no empty message shells
	t.Run("NoEmptyShells", func(t *testing.T) {
		emptyCount := 0
		for i, msg := range compactedMessages {
			if msg.Content == "" && msg.Thinking == "" && len(msg.ToolCalls) == 0 {
				t.Errorf("Found empty message at index %d", i)
				emptyCount++
			}
		}
		if emptyCount > 0 {
			t.Errorf("Found %d empty message shells (should be 0)", emptyCount)
		}
	})

	// TEST 2: Verify placeholders exist
	t.Run("PlaceholdersCreated", func(t *testing.T) {
		placeholderCount := 0
		for _, msg := range compactedMessages {
			if isPlaceholder(msg.Content) {
				placeholderCount++
			}
		}
		if placeholderCount == 0 {
			t.Error("No placeholders created - compaction may not have run")
		}
		t.Logf("Created %d placeholders", placeholderCount)
	})

	// TEST 3: Verify space savings
	t.Run("SpaceSavings", func(t *testing.T) {
		if compactedSize >= originalSize {
			t.Errorf("Compaction did not reduce size: before=%d after=%d",
				originalSize, compactedSize)
		}

		reductionPercent := float64(originalSize-compactedSize) / float64(originalSize) * 100
		if reductionPercent < 50 {
			t.Errorf("Compaction saved only %.1f%% (expected >50%%)", reductionPercent)
		}
	})

	// TEST 4: Verify message count reduction
	t.Run("MessageCountReduction", func(t *testing.T) {
		if compactedCount >= originalCount {
			t.Errorf("Message count did not decrease: before=%d after=%d",
				originalCount, compactedCount)
		}

		removed := originalCount - compactedCount
		if removed < 10 {
			t.Errorf("Only removed %d messages (expected significant reduction)", removed)
		}
	})
}

// TestCompactionPreservesToolCalls verifies tool calls are stored in database
func TestCompactionPreservesToolCalls(t *testing.T) {
	t.Skip("TODO: Implement after adding original_messages column to database schema")

	// This test will verify:
	// 1. Tool calls are stored in original_messages JSON
	// 2. Tool arguments are preserved
	// 3. Restoration recreates full message chain with tool calls
}

// TestEmptyMessageRemoval tests that compaction removes empty message shells
func TestEmptyMessageRemoval(t *testing.T) {
	// Create test messages with a turn
	messages := []state.Message{
		{Role: "system", Content: "System prompt"},
		{Role: "user", Content: "Do something"},
		{
			Role:    "assistant",
			Content: "Let me check",
			ToolCalls: []state.ToolCall{{
				ID:   "call-1",
				Type: "function",
				Function: state.FunctionCall{
					Name:      "read_file",
					Arguments: `{"path":"test.txt"}`,
				},
			}},
		},
		{
			Role:       "tool",
			Name:       "read_file",
			Content:    "File contents",
			ToolCallID: "call-1",
		},
		{
			Role:    "assistant",
			Content: "Done",
		},
	}

	// Setup
	ctx := context.Background()
	cfg := config.Config{
		MemoryStorePath:       filepath.Join(t.TempDir(), "test.db"),
		ContextMessagePercent: 0.02,
		ContextTotalPercent:   0.01, // Very low to force compaction
		ContextProtectRecent:  1,    // Protect only last 1 message
	}

	mockClient := &mockLLMClient{summaries: make(map[string]string)}
	profile, err := newMemoryProfile(Dependencies{
		Client:   mockClient,
		Config:   cfg,
		Provider: "test",
		Model:    "test-model",
	})
	if err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}
	defer profile.store.Close()

	// Set tool definitions (required for size calculation)
	profile.SetToolDefinitions([]tooling.ToolDefinition{})

	conv := newTestConversation(messages)

	// Run compaction
	prepared, err := profile.Prepare(ctx, conv)
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}

	// Verify no empty shells
	for i, msg := range prepared.Messages {
		if msg.Content == "" && msg.Thinking == "" && len(msg.ToolCalls) == 0 {
			t.Errorf("Found empty message shell at index %d", i)
		}
	}
}

// Mock LLM client for testing
type mockLLMClient struct {
	summaries map[string]string
}

func (m *mockLLMClient) Chat(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	// Generate a simple summary
	summary := "Test summary of conversation turn"
	m.summaries[req.Messages[len(req.Messages)-1].Content] = summary

	return llm.ChatResponse{
		Choices: []llm.ChatChoice{
			{Message: state.Message{Content: summary}},
		},
	}, nil
}

func (m *mockLLMClient) Name() string {
	return "mock"
}

// newTestConversation creates a Conversation for testing using reflection
func newTestConversation(messages []state.Message) *state.Conversation {
	conv := &state.Conversation{}

	// Use reflection to set private fields
	v := reflect.ValueOf(conv).Elem()

	// Set messages field
	messagesField := v.FieldByName("messages")
	reflect.NewAt(messagesField.Type(), unsafe.Pointer(messagesField.UnsafeAddr())).
		Elem().Set(reflect.ValueOf(messages))

	// Set other fields to reasonable defaults
	keyField := v.FieldByName("key")
	reflect.NewAt(keyField.Type(), unsafe.Pointer(keyField.UnsafeAddr())).
		Elem().SetString("test-conversation")

	createdField := v.FieldByName("createdAt")
	reflect.NewAt(createdField.Type(), unsafe.Pointer(createdField.UnsafeAddr())).
		Elem().Set(reflect.ValueOf(time.Now()))

	updatedField := v.FieldByName("updatedAt")
	reflect.NewAt(updatedField.Type(), unsafe.Pointer(updatedField.UnsafeAddr())).
		Elem().Set(reflect.ValueOf(time.Now()))

	return conv
}
