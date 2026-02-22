# Getting Started with tada Development

## Prerequisites

- Go 1.25.7 or later
- OpenAI API key

## Setup

1. Clone the repository:
```bash
git clone https://github.com/Lin-Jiong-HDU/tada.git
cd tada
```

2. Install dependencies:
```bash
go mod download
```

3. Configure API key:

**For OpenAI:**
```bash
mkdir -p ~/.tada
cat > ~/.tada/config.yaml << EOF
ai:
  provider: openai
  api_key: YOUR_OPENAI_API_KEY
  model: gpt-4o-mini
  base_url: https://api.openai.com/v1
EOF
```

**For GLM (Zhipu AI):**
```bash
mkdir -p ~/.tada
cat > ~/.tada/config.yaml << EOF
ai:
  provider: glm
  api_key: YOUR_GLM_API_KEY
  model: glm-5
  base_url: https://open.bigmodel.cn/api
EOF
```

## Running

```bash
go run cmd/tada/main.go "your request here"
```

## Testing

```bash
# Unit tests
go test ./...

# Integration tests (requires API key)
TADA_INTEGRATION_TEST=1 OPENAI_API_KEY=your-key go test ./...
```

## Async Execution

For long-running commands, use async mode:

```bash
tada "compile project &"
```

The command will be queued. Use `tada tasks` to:

1. View pending commands
2. Authorize (a) or reject (r) individual commands
3. Authorize all (A) or reject all (R)

Authorized tasks execute immediately. For batch execution, use:

```bash
tada run
```

This executes all approved tasks that haven't been run yet.

### Async Workflow

```bash
# Queue multiple async commands
tada "download data &"
tada "process data &"
tada "generate report &"

# Review and authorize in TUI
tada tasks

# Or batch execute later
tada run
```

## Chat Mode

For interactive conversations with AI, use chat mode:

```bash
# Start a new conversation
tada chat

# Use a specific prompt template
tada chat --prompt coder

# Resume a previous conversation
tada chat --continue <conversation-id>

# List all conversations
tada chat --list

# Show conversation details
tada chat --show <conversation-id>

# Delete a conversation
tada chat --delete <conversation-id>
```

### Chat Commands

While in chat mode, you can use these commands:

- `/help` - Show available commands
- `/clear` - Clear the screen
- `/prompt <name>` - Switch to a different prompt template
- `/exit` or `/quit` - Exit and save the conversation

### Prompt Templates

tada comes with built-in prompt templates:

- `default` - Friendly AI assistant for general questions
- `coder` - Programming assistant with code examples
- `expert` - Technical expert for deep analysis

You can create custom prompts by adding markdown files to `~/.tada/prompts/`:

```markdown
---
name: "my-prompt"
title: "My Custom Prompt"
description: "A custom assistant"
---

You are a specialized assistant for...
```

### Chat Configuration

Add chat settings to your `~/.tada/config.yaml`:

```yaml
chat:
  default_prompt: "default"      # Default prompt template
  max_history: 100               # Maximum messages per conversation
  auto_save: true                # Automatically save conversations
  stream: true                   # Enable streaming responses
  render_markdown: true          # Render markdown output
```

## Security Configuration

tada includes security controls to protect against dangerous AI-generated commands.

### Security Levels

```yaml
security:
  # always: every command requires confirmation
  # dangerous: only dangerous commands require confirmation (default)
  # never: no confirmation required
  command_level: dangerous

  # Paths that are completely forbidden
  restricted_paths:
    - /etc
    - /usr/bin
    - /System

  # Paths that cannot be written to
  readonly_paths:
    - ~/.ssh
    - ~/.gnupg

  # Allow shell commands (pipes, redirects)
  allow_shell: true

  # Allow terminal takeover (multi-step operations)
  allow_terminal_takeover: true
```

### Examples

```yaml
# Balanced mode (default)
security:
  command_level: dangerous
  allow_shell: true

# Paranoid mode
security:
  command_level: always
  allow_shell: false
  restricted_paths:
    - /etc
    - /usr
    - /System
    - /home
```

## Project Structure

See `docs/plans/2025-02-18-tada-mvp.md` for implementation details.

```
tada/
├── cmd/
│   └── tada/
│       └── main.go          # CLI entry point
├── internal/
│   ├── ai/
│   │   ├── provider.go      # AI types and interfaces
│   │   ├── openai/          # OpenAI implementation
│   │   └── glm/             # GLM implementation
│   ├── core/
│   │   ├── engine.go        # Main orchestration
│   │   ├── executor.go      # Command execution
│   │   └── queue/           # Task queue management
│   ├── conversation/        # Chat conversation features
│   │   ├── types.go         # Conversation types
│   │   ├── storage.go       # Conversation persistence
│   │   ├── manager.go       # Conversation manager
│   │   ├── prompt.go        # Prompt template loader
│   │   └── renderer.go      # Markdown renderer
│   ├── terminal/
│   │   └── repl.go          # Interactive REPL
│   └── storage/
│       ├── config.go        # Configuration management
│       └── session.go       # Session persistence
├── tests/
│   └── integration/         # Integration tests
└── docs/
    ├── plans/
    │   └── 2025-02-18-tada-mvp.md
    └── getting-started.md
```
