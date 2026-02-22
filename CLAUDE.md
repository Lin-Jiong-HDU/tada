# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`tada` is a terminal AI assistant written in Go that understands natural language and executes commands. Key features include async command queuing, security controls, and TUI-based task authorization.

**Architecture**: CLI → Engine (orchestration) → AI Provider + Security + Executor + Queue

## Common Commands

```bash
# Build
go build -o tada cmd/tada/main.go

# Run
go run cmd/tada/main.go "your request here"
./tada "list files"

# Test
go test ./...                          # All tests
go test ./internal/core -v             # Specific package
go test ./internal/core -run TestEngine -v  # Specific test

# Integration tests (requires API key)
TADA_INTEGRATION_TEST=1 go test ./... -v

# Format
go fmt ./...

# Vet
go vet ./...
```

## Project Architecture

### Layer Structure

```
cmd/tada/          - CLI entry point (cobra commands)
internal/
├── ai/            - AI provider interface (openai, glm implementations)
├── core/
│   ├── engine.go  - Main orchestration (AI → Security → Executor)
│   ├── queue/     - Task queue for async execution
│   ├── tui/       - Bubble Tea TUI for task authorization
│   └── security/  - Security controller (danger check, path access, shell analysis)
├── storage/       - Config and session persistence (~/.tada)
└── terminal/      - Terminal prompts and confirmations
```

### Key Data Flow

**Sync Command**: `tada "create folder"` → Engine.Process() → AI ParseIntent → Security Check → User Confirm → Execute → Output Analysis

**Async Command**: `tada "compile &"` → ParseAsyncSyntax detects `&` → Set IsAsync flag → Queue task (no confirm) → `tada tasks` TUI to authorize → Execute on approval

### Critical Components

**Engine** (`internal/core/engine.go`): Orchestrates the entire workflow. Key methods:

- `Process()`: Main entry point, handles sync/async, security, execution
- `ParseAsyncSyntax()`: Detects `&` suffix for async mode
- `StripAsyncSyntax()`: Removes `&` from input

**Queue** (`internal/core/queue/`): Task management with state machine:

- States: pending → approved → executing → completed/failed
- `Manager`: Queue persistence and task lifecycle
- `TaskExecutor`: Executes approved tasks

**Security** (`internal/core/security/`): Multi-layered checks:

- `DangerousCommandChecker`: Built-in dangerous command list (rm -rf, etc.)
- `PathAccessChecker`: Restricts access to sensitive paths
- `ShellCommandAnalyzer`: Detects dangerous shell operations (pipes, redirects)
- Policy levels: always/dangerous/never confirmation

**TUI** (`internal/core/tui/`): Bubble Tea interface for task authorization:

- `queue_model.go`: Model with vim-style navigation (j/k, gg, G)
- `keys.go`: Key bindings and help text
- Status bar fixed at window bottom with dynamic padding

## Important Implementation Details

### Async Execution Flow

1. User appends `&` to command: `tada "download file &"`
2. `ParseAsyncSyntax()` detects the suffix, sets `IsAsync=true`
3. Commands are queued WITHOUT confirmation
4. User runs `tada tasks` to open TUI
5. In TUI, pressing `a` authorizes AND executes immediately
6. Tasks show status indicators: ⋯ (executing), ✓ (completed), ✗ (failed)

### Session Management

- Each CLI run creates/reuses a session in `~/.tada/sessions/<session-id>/`
- Queue file: `~/.tada/sessions/<session-id>/queue.json`
- Session persistence enables async task review across invocations

### TUI Key Bindings

- `↑/k`: Up, `↓/j`: Down, `gg`: Top, `G`: Bottom
- `a`: Authorize and execute current task
- `r`: Reject current task
- `A`: Execute all approved, `R`: Reject all
- `q`: Quit

## Configuration

Location: `~/.tada/config.yaml`

```yaml
ai:
  provider: openai # or glm/zhipu
  api_key: sk-xxx
  model: gpt-4o-mini
  base_url: https://api.openai.com/v1

security:
  command_level: dangerous # always | dangerous | never
  restricted_paths: [] # Forbidden paths
  readonly_paths: [] # Read-only paths
  allow_shell: true
```

## Testing Patterns

- Unit tests alongside source files (`*_test.go`)
- Integration tests in `tests/integration/`
- E2E tests in `tests/e2e/` (require `TADA_INTEGRATION_TEST=1`)
- Mock AI providers for testing without API calls

## CLI Commands

- `tada <prompt>` - Sync command (default)
- `tasks` - Open TUI to review/authorize queued tasks
- `run` - Execute all approved tasks in batch
- `-i` flag - Incognito mode (no history saved)
