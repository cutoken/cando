package agent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type inputHistory struct {
	path    string
	entries []string
	total   int
	mu      sync.Mutex
}

func loadInputHistory(path string) *inputHistory {
	h := &inputHistory{path: path}
	if path == "" {
		return h
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return h
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r\n")
		if strings.TrimSpace(line) == "" {
			continue
		}
		h.entries = append(h.entries, line)
		h.total += len(line)
	}
	return h
}

func (h *inputHistory) Entries() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	cpy := make([]string, len(h.entries))
	copy(cpy, h.entries)
	return cpy
}

func (h *inputHistory) Add(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, line)
	h.total += len(line)
	if h.path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(h.path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(h.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintln(f, line)
}

func (h *inputHistory) Stats() (count, chars int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.entries), h.total
}
