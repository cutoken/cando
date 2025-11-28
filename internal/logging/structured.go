package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"ts"`
	Level     string                 `json:"level"`
	Component string                 `json:"component,omitempty"`
	Workspace string                 `json:"workspace,omitempty"`
	Message   string                 `json:"msg"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// StructuredLogger wraps a standard logger with structured logging
type StructuredLogger struct {
	logger    *log.Logger
	component string
	workspace string
	jsonMode  bool
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(logger *log.Logger, component string, jsonMode bool) *StructuredLogger {
	return &StructuredLogger{
		logger:    logger,
		component: component,
		jsonMode:  jsonMode,
	}
}

// WithWorkspace returns a logger with workspace context
func (s *StructuredLogger) WithWorkspace(workspace string) *StructuredLogger {
	return &StructuredLogger{
		logger:    s.logger,
		component: s.component,
		workspace: workspace,
		jsonMode:  s.jsonMode,
	}
}

// WithComponent returns a logger with component context
func (s *StructuredLogger) WithComponent(component string) *StructuredLogger {
	return &StructuredLogger{
		logger:    s.logger,
		component: component,
		workspace: s.workspace,
		jsonMode:  s.jsonMode,
	}
}

// log formats and writes the log entry
func (s *StructuredLogger) log(level string, msg string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Component: s.component,
		Workspace: s.workspace,
		Message:   msg,
		Fields:    fields,
	}

	if s.jsonMode {
		// JSON mode for structured parsing
		data, _ := json.Marshal(entry)
		s.logger.Println(string(data))
	} else {
		// Human-readable format
		prefix := ""
		if s.component != "" {
			prefix = fmt.Sprintf("[%s] ", s.component)
		}
		if s.workspace != "" {
			prefix += fmt.Sprintf("[ws:%s] ", s.workspace)
		}
		
		output := prefix + msg
		if len(fields) > 0 {
			output += " |"
			for k, v := range fields {
				output += fmt.Sprintf(" %s=%v", k, v)
			}
		}
		s.logger.Println(output)
	}
}

// Info logs an info message
func (s *StructuredLogger) Info(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields...)
	s.log("INFO", msg, f)
}

// Error logs an error message
func (s *StructuredLogger) Error(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields...)
	s.log("ERROR", msg, f)
}

// Debug logs a debug message
func (s *StructuredLogger) Debug(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields...)
	s.log("DEBUG", msg, f)
}

// Warn logs a warning message
func (s *StructuredLogger) Warn(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields...)
	s.log("WARN", msg, f)
}

// Printf provides compatibility with standard logger interface
func (s *StructuredLogger) Printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.Info(msg)
}

// mergeFields combines multiple field maps
func mergeFields(fields ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range fields {
		for k, v := range m {
			result[k] = v
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}