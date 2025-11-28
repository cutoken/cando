# Logging Strategy for Cando

This document outlines the strategic logging improvements implemented to help both users and developers.

## Logging Framework

A new logging package (`internal/logging`) provides three levels of logging:

### 1. UserLog - Always Visible
Important user-facing information that helps users understand what's happening:
- Session creation/resumption
- Configuration reloads
- Tool execution starts
- Memory pin/unpin operations

### 2. ErrorLog - Always Visible  
Errors and critical failures:
- API errors from providers
- Tool execution failures
- Session management errors
- JSON parsing failures

### 3. DevLog - DEV_MODE Only
Detailed debugging information for development:
- Request/response details
- Token usage metrics
- Tool execution timing
- Memory operations details
- Provider communication details

Enable with: `DEV_MODE=1 ./cando`

## Key Logging Locations

### Agent Operations (`internal/agent/agent.go`)
- **User prompts**: Input length and response status
- **Provider calls**: Message count, context size, token usage
- **Tool execution**: Tool name, timing, success/failure
- **Session management**: Creation, resumption, errors
- **Configuration**: Reload status and errors

### Provider Clients
- **OpenRouter** (`internal/openrouter/client.go`): Request/response details, API errors
- **Z.AI** (`internal/zai/client.go`): Enhanced response parsing, thinking mode status

### Tool Execution (`internal/tooling/tool.go`)
- **Shell Tool**: Command execution, working directory, timing, exit codes
- **Security**: Blocked command attempts
- **Performance**: Command duration and timeout handling

### Memory Operations (`internal/contextprofile/memory_tools.go`)
- **Recall**: Memory ID access, expansion success/failure
- **Pin/Unpin**: Memory ID, pin status, pinned count
- **Errors**: Unmarshal failures, access errors

## Usage Examples

### Development Mode
```bash
DEV_MODE=1 ./cando
# Shows detailed debugging information
```

### Production Mode (default)
```bash
./cando
# Shows only user-facing information and errors
```

## Log Format

All logs follow consistent formats:
- `[USER] message` - User-facing information
- `[ERROR] message` - Errors and critical failures  
- `[DEV] message` - Development debugging (DEV_MODE only)

## Benefits

### For Users
- Clear feedback on long-running operations
- Transparency about tool execution
- Session state visibility
- Configuration change confirmation

### For Developers
- Detailed request/response tracing
- Performance metrics and timing
- Error context and debugging info
- Provider communication details

### For Support
- Structured log levels for filtering
- Consistent format for log parsing
- Error correlation across components
- Operational visibility

## Implementation Notes

- All logging is non-blocking and won't impact performance
- DEV_MODE logging can be enabled without recompilation
- Sensitive information (API keys, tokens) is never logged
- Log levels are designed to minimize noise while maximizing utility