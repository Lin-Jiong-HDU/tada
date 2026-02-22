# tada 🪄

> **"Tada! And it's done."**

`tada` is a lightweight terminal AI assistant written in Go. It understands your intent and automatically executes commands, freeing you from tedious CLI syntax.

## Features

- 🗣️ Natural language interface - just tell it what you want
- 🧠 AI-powered intent understanding
- 💬 Interactive chat mode with conversation history
- 💾 Session persistence with history
- 🔒 Security controls - dangerous command detection and path access control
- 🛡️ Configurable security levels (always/dangerous/never confirmation)
- 📝 Custom prompt templates for different use cases

## Installation

```bash
git clone https://github.com/Lin-Jiong-HDU/tada.git
cd tada
go build -o tada cmd/tada/main.go
sudo mv tada /usr/local/bin/
```

## Configuration

Create `~/.tada/config.yaml`:

**For OpenAI:**
```yaml
ai:
  provider: openai
  api_key: sk-xxx  # Your OpenAI API key
  model: gpt-4o-mini
  base_url: https://api.openai.com/v1
  timeout: 30
  max_tokens: 4096
```

**For GLM (Zhipu AI):**
```yaml
ai:
  provider: glm
  api_key: xxx  # Your GLM API key
  model: glm-5
  base_url: https://open.bigmodel.cn/api
  timeout: 30
  max_tokens: 4096
```

**Security Configuration (optional):**
```yaml
security:
  command_level: dangerous    # always | dangerous | never
  restricted_paths:            # Forbidden paths
    - /etc
    - /usr/bin
  readonly_paths:              # Read-only paths
    - ~/.ssh
    - ~/.gnupg
  allow_shell: true            # Allow shell commands
```

## Security

tada includes built-in security controls to protect against dangerous AI-generated commands:

- 🔒 **Dangerous command detection** - Built-in list + AI judgment
- 🛡️ **Path access control** - Restrict access to sensitive paths
- 📝 **Read-only protection** - Protect important files from modification
- 🔧 **Shell analysis** - Detect potentially dangerous shell operations

## Usage

```bash
# Simple commands (synchronous)
tada "list all files in the current directory"
tada "create a new folder named docs"

# Interactive chat mode
tada chat                    # Start a new conversation
tada chat --prompt coder      # Use a specific prompt template
tada chat --continue <id>     # Resume a previous conversation
tada chat --list              # List all conversations
tada chat --show <id>         # Show conversation details

# Async commands (queue for later execution)
tada "create a new folder named tmp &"
tada "download large file &"

# View and authorize pending tasks
tada tasks

# Execute all approved tasks
tada run

# Incognito mode (no history saved)
tada -i "run a secret command"
```

### Chat Mode

Start an interactive conversation with AI:

```bash
# Start a new conversation
tada chat

# Use a specific prompt template
tada chat --prompt coder

# Resume a conversation
tada chat --continue <conversation-id>

# List all conversations
tada chat --list
```

**Available Commands in Chat:**
- `/help` - Show help
- `/clear` - Clear screen
- `/prompt <name>` - Switch prompt template
- `/exit` or `/quit` - Exit and save

**Available Prompt Templates:**
- `default` - Friendly AI assistant
- `coder` - Programming assistant
- `expert` - Technical expert

### Async Execution

Add `&` at the end of your command to run it asynchronously:

```bash
tada "long running task &"
```

Async commands are queued without immediate confirmation. Use `tada tasks` to review and authorize them. Authorized tasks execute immediately in the TUI, or use `tada run` for batch execution.

## Development

See [docs/getting-started.md](docs/getting-started.md) for development setup.

## Roadmap

- [x] Phase 1: MVP (CLI + AI + Command Execution)
- [x] Phase 2: Security Controls
- [ ] Phase 3: TUI + Authorization UI
- [ ] Phase 4: Multiple AI Providers + i18n

## License

MIT
