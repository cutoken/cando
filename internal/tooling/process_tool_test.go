package tooling

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackgroundProcessToolStartListLogs(t *testing.T) {
	workdir := t.TempDir()
	processDir := filepath.Join(t.TempDir(), "processes")
	guard, err := newPathGuard(workdir)
	if err != nil {
		t.Fatalf("newPathGuard: %v", err)
	}
	tool := NewBackgroundProcessTool(guard, processDir, "")

	resp, err := tool.Call(context.Background(), map[string]any{
		"action":  "start",
		"command": []string{"/bin/sh", "-c", "printf 'hello world'\n"},
	})
	if err != nil {
		t.Fatalf("start call failed: %v", err)
	}
	jobID := parseJobID(t, resp)

	metaPath := filepath.Join(processDir, jobID, "meta.json")
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("meta not created: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	listResp, err := tool.Call(context.Background(), map[string]any{
		"action": "list",
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !strings.Contains(listResp, jobID) {
		t.Fatalf("list output missing job: %s", listResp)
	}

	logResp, err := tool.Call(context.Background(), map[string]any{
		"action":     "logs",
		"job_id":     jobID,
		"tail_lines": 5,
	})
	if err != nil {
		t.Fatalf("logs failed: %v", err)
	}
	if !strings.Contains(logResp, "hello world") {
		t.Fatalf("logs missing expected output: %s", logResp)
	}
}

func TestBackgroundProcessToolKill(t *testing.T) {
	workdir := t.TempDir()
	processDir := filepath.Join(t.TempDir(), "processes")
	guard, err := newPathGuard(workdir)
	if err != nil {
		t.Fatalf("newPathGuard: %v", err)
	}
	tool := NewBackgroundProcessTool(guard, processDir, "")

	resp, err := tool.Call(context.Background(), map[string]any{
		"action":  "start",
		"command": []string{"/bin/sh", "-c", "sleep 5"},
	})
	if err != nil {
		t.Fatalf("start call failed: %v", err)
	}
	jobID := parseJobID(t, resp)

	killResp, err := tool.Call(context.Background(), map[string]any{
		"action": "kill",
		"job_id": jobID,
	})
	if err != nil {
		t.Fatalf("kill failed: %v", err)
	}
	if !strings.Contains(killResp, "\"status\":\"killed\"") {
		t.Fatalf("kill response unexpected: %s", killResp)
	}

	time.Sleep(100 * time.Millisecond)
	metaPath := filepath.Join(processDir, jobID, "meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	if !strings.Contains(string(data), "\"status\"") {
		t.Fatalf("meta missing status: %s", string(data))
	}
}

func parseJobID(t *testing.T, payload string) string {
	t.Helper()
	var obj map[string]any
	if err := json.Unmarshal([]byte(payload), &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	id, _ := obj["job_id"].(string)
	if id == "" {
		t.Fatalf("job_id missing in %s", payload)
	}
	return id
}
