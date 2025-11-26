package prompts

import (
	_ "embed"
	"strings"
	"sync"
)

//go:embed system_cando_compact.txt
var baseSystemPrompt string

var (
	metadataMu sync.RWMutex
	metadata   string
)

// Base returns the built-in Cando system prompt.
func Base() string {
	return strings.TrimSpace(baseSystemPrompt)
}

// Combine joins the built-in prompt with an optional user-provided prompt.
func Combine(user string) string {
	base := Base()
	trimmed := strings.TrimSpace(user)
	var sections []string
	sections = append(sections, base)

	if meta := getMetadata(); meta != "" {
		sections = append(sections, "## Environment Context\n"+meta)
	}

	if trimmed != "" {
		sections = append(sections, trimmed)
	}

	return strings.Join(sections, "\n\n")
}

// ExtractUserPortion strips the base prompt and environment context from a combined prompt,
// returning only the user's custom portion. If the input doesn't contain the base prompt,
// returns the input unchanged.
func ExtractUserPortion(combined string) string {
	combined = strings.TrimSpace(combined)
	if combined == "" {
		return ""
	}

	base := Base()

	// If the combined text starts with the base prompt, strip it
	if strings.HasPrefix(combined, base) {
		remaining := strings.TrimSpace(combined[len(base):])

		// Also strip environment context section if present
		envHeader := "## Environment Context"
		if idx := strings.Index(remaining, envHeader); idx == 0 {
			// Find the end of the environment section (next ## header or end of string)
			afterHeader := remaining[len(envHeader):]
			if nextSection := strings.Index(afterHeader, "\n##"); nextSection != -1 {
				remaining = strings.TrimSpace(afterHeader[nextSection:])
			} else {
				// Environment context is at the end, look for double newline
				parts := strings.SplitN(remaining, "\n\n", 2)
				if len(parts) > 1 {
					remaining = strings.TrimSpace(parts[1])
				} else {
					remaining = ""
				}
			}
		}

		return remaining
	}

	// Input doesn't start with base prompt, return as-is
	return combined
}

// SetMetadata defines the environment metadata appended to the system prompt.
func SetMetadata(info string) {
	metadataMu.Lock()
	defer metadataMu.Unlock()
	metadata = strings.TrimSpace(info)
}

func getMetadata() string {
	metadataMu.RLock()
	defer metadataMu.RUnlock()
	return metadata
}
