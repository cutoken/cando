package llm

import (
	"errors"
	"fmt"
	"time"
)

// ErrorType classifies provider errors for UI handling
type ErrorType string

const (
	ErrorTypeRateLimit          ErrorType = "rate_limit"          // 429 - too many requests
	ErrorTypeQuotaExceeded      ErrorType = "quota_exceeded"      // Usage limit (ZAI 1308)
	ErrorTypeInsufficientCredit ErrorType = "insufficient_credit" // 402 - no balance
	ErrorTypeProviderDown       ErrorType = "provider_down"       // 502/503 - upstream issue
	ErrorTypeAuth               ErrorType = "auth"                // 401 - bad API key
	ErrorTypeModeration         ErrorType = "moderation"          // 403 - content flagged
	ErrorTypeUnknown            ErrorType = "unknown"             // Fallback
)

// ProviderError is a structured error returned by LLM clients
type ProviderError struct {
	Type       ErrorType      // Classification
	Provider   string         // "zai", "openrouter"
	Code       string         // Raw error code ("1308", "429")
	Message    string         // Human-readable message
	ResetAt    *time.Time     // When limit resets (if known)
	RetryAfter *time.Duration // How long to wait (if known)
	Retryable  bool           // Should we auto-retry?
}

func (e *ProviderError) Error() string {
	if e.ResetAt != nil {
		return fmt.Sprintf("%s: %s (resets at %s)", e.Provider, e.Message, e.ResetAt.Format("15:04:05"))
	}
	return fmt.Sprintf("%s: %s", e.Provider, e.Message)
}

// Unwrap allows errors.Is/As to work through wrapped errors
func (e *ProviderError) Unwrap() error {
	return nil
}

// IsProviderError checks if err is a ProviderError and returns it
func IsProviderError(err error) (*ProviderError, bool) {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe, true
	}
	return nil, false
}

// NewProviderError creates a new ProviderError with the given parameters
func NewProviderError(provider string, errType ErrorType, code, message string) *ProviderError {
	return &ProviderError{
		Type:     errType,
		Provider: provider,
		Code:     code,
		Message:  message,
	}
}
