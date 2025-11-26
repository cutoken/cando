package tooling

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestApplyPatchToolUpdate(t *testing.T) {
	dir := t.TempDir()
	guard, err := newPathGuard(dir)
	if err != nil {
		t.Fatalf("guard: %v", err)
	}
	tool := NewApplyPatchTool(guard)

	orig := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(orig, []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatalf("write orig: %v", err)
	}

	patch := `*** Begin Patch
*** Update File: file.txt
@@ -1 +1 @@
-hello
+hi
*** End Patch`

	if _, err := tool.Call(context.Background(), map[string]any{"patch": patch}); err != nil {
		t.Fatalf("apply patch: %v", err)
	}

	got, err := os.ReadFile(orig)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "hi\nworld\n" {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestApplyPatchToolAddDelete(t *testing.T) {
	dir := t.TempDir()
	guard, err := newPathGuard(dir)
	if err != nil {
		t.Fatalf("guard: %v", err)
	}
	tool := NewApplyPatchTool(guard)

	patch := `*** Begin Patch
*** Add File: foo.txt
+line1
+line2
*** End Patch

*** Begin Patch
*** Delete File: foo.txt
*** End Patch`

	if _, err := tool.Call(context.Background(), map[string]any{"patch": patch}); err != nil {
		t.Fatalf("apply patch: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "foo.txt")); !os.IsNotExist(err) {
		t.Fatalf("file should be deleted, err=%v", err)
	}
}
