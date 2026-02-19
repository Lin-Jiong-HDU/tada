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
│   │   └── openai/
│   │       ├── client.go    # OpenAI implementation
│   │       └── client_test.go
│   ├── core/
│   │   ├── engine.go        # Main orchestration
│   │   ├── executor.go      # Command execution
│   │   └── executor_test.go
│   └── storage/
│       ├── config.go        # Configuration management
│       ├── config_test.go
│       ├── session.go       # Session persistence
│       └── session_test.go
├── tests/
│   └── e2e/
│       └── mvp_test.go      # End-to-end tests
└── docs/
    ├── plans/
    │   └── 2025-02-18-tada-mvp.md
    └── getting-started.md
```
