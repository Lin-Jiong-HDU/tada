# tada 🪄

> **"Tada! And it's done."**

`tada` is a lightweight terminal AI assistant written in Go. It understands your intent and automatically executes commands, freeing you from tedious CLI syntax.

## Features

- 🗣️ Natural language interface - just tell it what you want
- 🧠 AI-powered intent understanding
- 💾 Session persistence with history
- 🔒 Configurable security levels (coming in Phase 2)
- 🔌 Plugin system (coming in Phase 2)

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

## Usage

```bash
# Simple commands
tada "list all files in the current directory"
tada "create a new folder named docs"
tada "say hello to the world"

# Incognito mode (no history saved)
tada -i "run a secret command"
```

## Development

See [docs/getting-started.md](docs/getting-started.md) for development setup.

## Roadmap

- [x] Phase 1: MVP (CLI + AI + Command Execution)
- [ ] Phase 2: Plugins + Security
- [ ] Phase 3: TUI + Async Tasks
- [ ] Phase 4: Multiple AI Providers + i18n

## License

MIT
