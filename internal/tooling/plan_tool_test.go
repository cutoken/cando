package tooling

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanToolUpdateGetHistory(t *testing.T) {
	root := t.TempDir()
	planPath := filepath.Join(root, "plan.json")
	guard, err := newPathGuard(root)
	if err != nil {
		t.Fatalf("newPathGuard: %v", err)
	}
	tool := NewPlanToolWithGuard(planPath, guard)

	if _, err := tool.Call(context.Background(), map[string]any{
		"action": "update",
		"steps": []any{
			map[string]any{"status": "pending", "step": "write tests"},
		},
	}); err != nil {
		t.Fatalf("update call failed: %v", err)
	}
	if _, err := os.Stat(planPath); err != nil {
		t.Fatalf("plan file not created: %v", err)
	}

	getResp, err := tool.Call(context.Background(), map[string]any{
		"action": "get",
	})
	if err != nil {
		t.Fatalf("get call failed: %v", err)
	}
	if !jsonContains(getResp, "write tests") {
		t.Fatalf("get response missing step: %s", getResp)
	}

	historyResp, err := tool.Call(context.Background(), map[string]any{
		"action": "history",
		"limit":  5,
	})
	if err != nil {
		t.Fatalf("history failed: %v", err)
	}
	if !jsonContains(historyResp, "write tests") {
		t.Fatalf("history response missing step: %s", historyResp)
	}
}

func jsonContains(payload string, needle string) bool {
	return strings.Contains(payload, needle)
}
