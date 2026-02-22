# Tada Chat Conversation Feature Design

> **创建日期:** 2026-02-22
> **状态:** 设计已完成，待实现

## 1. 目标

为 tada 添加纯对话功能，与现有的命令执行功能分离。支持多轮对话、历史持久化、自定义 prompt 模板、流式输出和 markdown 终端渲染。

## 2. 需求概述

### 2.1 核心功能
- **纯对话模式**: `tada chat` 仅用于对话，不执行命令
- **交互式 REPL**: 类似 ChatGPT 的多轮对话体验
- **对话恢复**: 退出后显示对话 ID，可通过 ID 恢复对话
- **历史持久化**: 对话保存到 `~/.tada/conversations/YYYYMMDD/<id>/`
- **自定义角色**: 支持多个 prompt 模板，可配置选择
- **流式输出**: AI 响应实时显示，提升交互体验
- **Markdown 渲染**: 在终端中美观显示 markdown 格式输出

### 2.2 用户场景

```bash
# 启动新对话（默认 prompt）
$ tada chat

# 指定 prompt 模板
$ tada chat --prompt coder

# 恢复已有对话
$ tada chat --continue abc123-def456

# 列出所有对话
$ tada chat --list

# 列出今天的对话
$ tada chat --list --today
```

## 3. 架构设计

### 3.1 目录结构

```
tada/
├── internal/
│   ├── conversation/           # 新建：对话管理包
│   │   ├── manager.go          # 对话管理器
│   │   ├── storage.go          # 对话存储
│   │   ├── prompt.go           # Prompt 模板加载器
│   │   ├── renderer.go         # Markdown 渲染器
│   │   └── types.go            # 数据结构定义
│   ├── ai/
│   │   └── provider.go         # 扩展：添加 ChatStream 方法
│   └── terminal/
│       └── repl.go             # 新建：REPL 交互组件
├── cmd/tada/
│   └── chat.go                 # 重写：chat 命令
└── ~/.tada/
    ├── prompts/                # 新建：Prompt 模板
    │   ├── default.md
    │   ├── coder.md
    │   └── expert.md
    └── conversations/          # 新建：对话历史（按日期分组）
        └── YYYYMMDD/
            └── <conversation-id>/
                └── messages.json
```

### 3.2 存储结构详情

```
~/.tada/conversations/
├── 20260222/                # 2026年2月22日的对话
│   ├── abc123-def456/
│   │   └── messages.json
│   └── def456-ghi789/
│       └── messages.json
├── 20260221/                # 2026年2月21日的对话
│   └── xyz111-uvw222/
│       └── messages.json
└── 20260220/
    └── .../
```

### 3.3 数据流

```
用户输入 "hello"
    ↓
chatCmd 启动 REPL
    ↓
ConversationManager.LoadOrCreate(conversationID)
    ↓
PromptLoader.Load(promptName)
    ↓
AIProvider.ChatStream(messages)  // 流式输出
    ↓
流式显示原始响应
    ↓
响应完成 → Markdown 渲染美化显示
    ↓
ConversationStorage.Save() → ~/.tada/conversations/20260222/<id>/
    ↓
继续 REPL 循环
```

### 3.4 与现有组件的关系

```
现有组件                  新组件
┌─────────────┐         ┌──────────────┐
│  chatCmd    │────────>│    REPL      │
│ (重写)      │         │  (新建)      │
└─────────────┘         └──────┬───────┘
                               │
                        ┌──────▼───────┐
                        │   Manager    │
                        │  (新建)      │
                        └──────┬───────┘
                               │
            ┌──────────────────┼──────────────────┐
            ↓                  ↓                  ↓
    ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
    │ PromptLoader│   │  Storage    │  │  Renderer   │
    │  (新建)     │   │  (新建)     │  │  (新建)     │
    └─────────────┘   └─────────────┘   └─────────────┘
                                                  │
            ┌─────────────────────────────────────┘
            ↓
    ┌─────────────────┐
    │AIProvider       │
    │ChatStream(扩展) │
    └─────────────────┘
```

## 4. 数据结构

### 4.1 Conversation

```go
// internal/conversation/types.go

// Conversation 表示一个对话
type Conversation struct {
    ID          string       `json:"id"`           // UUID
    Name        string       `json:"name"`         // 可读名称
    PromptName  string       `json:"prompt_name"`  // 使用的 prompt 模板
    Messages    []Message    `json:"messages"`     // 消息历史
    CreatedAt   time.Time    `json:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at"`
}

// Message 表示单条消息
type Message struct {
    Role      string    `json:"role"`      // "system" | "user" | "assistant"
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
}
```

### 4.2 Prompt 模板

**文件格式** (`~/.tada/prompts/<name>.md`):

```markdown
---
name: "coder"
title: "编程助手"
description: "专业的编程对话助手"
---

你是一位经验丰富的程序员，擅长 Go、Python、JavaScript 等语言。
你的回答应该简洁、准确，提供可执行的代码示例。
```

**数据结构**:

```go
// internal/conversation/prompt.go

type PromptTemplate struct {
    Name         string  // 模板唯一标识
    Title        string  // 显示标题
    Description  string  // 描述
    Content      string  // 原始 markdown 内容
    SystemPrompt string  // 提取的 system prompt (--- 后的内容)
}
```

### 4.3 配置扩展

```yaml
# ~/.tada/config.yaml

ai:
  provider: openai
  api_key: sk-xxx
  model: gpt-4o-mini
  base_url: https://api.openai.com/v1

# 新增：chat 配置
chat:
  default_prompt: "default"    # 默认使用的 prompt
  max_history: 100             # 最大历史消息数
  auto_save: true              # 自动保存对话
  stream: true                 # 默认启用流式输出
  render_markdown: true        # 启用 markdown 渲染
```

## 5. 核心组件

### 5.1 ConversationManager

```go
// internal/conversation/manager.go

type Manager struct {
    storage     Storage
    promptLoader *PromptLoader
    aiProvider  ai.AIProvider
    config      *ChatConfig
}

// 核心方法
func (m *Manager) Create(name, promptName string) (*Conversation, error)
func (m *Manager) Get(id string) (*Conversation, error)
func (m *Manager) List() ([]*Conversation, error)
func (m *Manager) ListByDate(date string) ([]*Conversation, error)  // 按日期列出
func (m *Manager) Delete(id string) error
func (m *Manager) AppendMessage(convID string, msg Message) error
func (m *Manager) Chat(convID string, userInput string) (string, error)
func (m *Manager) ChatStream(convID string, userInput string) (<-chan string, error)
```

### 5.2 PromptLoader

```go
// internal/conversation/prompt.go

type PromptLoader struct {
    promptsDir string
}

func (l *PromptLoader) Load(name string) (*PromptTemplate, error)
func (l *PromptLoader) List() ([]*PromptTemplate, error)
func (l *PromptLoader) ExtractSystemPrompt(content string) string
```

### 5.3 ConversationStorage

```go
// internal/conversation/storage.go

type Storage interface {
    Save(conv *Conversation) error
    Load(id string) (*Conversation, error)
    List() ([]*Conversation, error)
    ListByDate(date string) ([]*Conversation, error)
    Delete(id string) error
}

type FileStorage struct {
    conversationsDir string  // ~/.tada/conversations
}

// GetDatePath 返回对话的日期路径
func (s *FileStorage) GetDatePath(conv *Conversation) string {
    date := conv.CreatedAt.Format("20060102")
    return filepath.Join(s.conversationsDir, date)
}

// GetConversationPath 返回对话的完整路径
func (s *FileStorage) GetConversationPath(convID string) string {
    // 需要先遍历日期文件夹查找，或在 Conversation 中存储日期信息
}
```

### 5.4 Renderer (Markdown 渲染器)

```go
// internal/conversation/renderer.go

type Renderer struct {
    glamourTerm *glamour.Term
}

// NewRenderer 创建 markdown 渲染器
func NewRenderer(width int) (*Renderer, error) {
    term, _ := glamour.NewTerm(
        glamour.WithAutoStyle(),
        glamour.WithWordWrap(width),
    )
    return &Renderer{glamourTerm: term}, nil
}

// Render 渲染 markdown 文本
func (r *Renderer) Render(markdown string) (string, error) {
    out, err := r.glamourTerm.Render(markdown)
    if err != nil {
        // 降级：渲染失败返回原始文本
        return markdown, nil
    }
    return out, nil
}
```

### 5.5 REPL

```go
// internal/terminal/repl.go

type REPL struct {
    manager      *conversation.Manager
    renderer     *Renderer
    conversation *Conversation
    stream       bool
    showThinking bool
}

func (r *REPL) Run(convID string) error
func (r *REPL) processInput(input string) error
func (r *REPL) displayStreamResponse(stream <-chan string)
func (r *REPL) displayRenderedResponse(markdown string)
func (r *REPL) displayExitSummary()
func (r *REPL) handleCommand(input string) (shouldExit bool, err error)
```

**REPL 命令**:
- `/exit`, `/quit` - 退出并保存
- `/help` - 显示帮助
- `/clear` - 清屏
- `/prompt <name>` - 切换 prompt
- `/save <name>` - 保存对话副本

## 6. AIProvider 接口扩展

```go
// internal/ai/provider.go

type AIProvider interface {
    // 现有方法
    ParseIntent(ctx context.Context, input string, systemPrompt string) (*Intent, error)
    AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error)
    Chat(ctx context.Context, messages []Message) (string, error)

    // 新增：流式对话
    ChatStream(ctx context.Context, messages []Message) (<-chan string, error)
}
```

### 6.1 OpenAI 实现示例

```go
// internal/ai/openai/client.go

func (c *Client) ChatStream(ctx context.Context, messages []Message) (<-chan string, error) {
    // 使用 SSE (Server-Sent Events)
    // 设置 stream: true
    // 返回 channel 逐块发送响应
}
```

## 7. 命令行接口

### 7.1 命令定义

```bash
# 启动新对话（默认 prompt）
tada chat

# 启动新对话（指定 prompt）
tada chat --prompt coder

# 恢复已有对话
tada chat --continue <conversation-id>

# 列出所有对话
tada chat --list

# 列出今天的对话
tada chat --list --today

# 列出指定日期的对话
tada chat --list --date 20260222

# 查看对话详情
tada chat --show <conversation-id>

# 删除对话
tada chat --delete <conversation-id>

# 不保存历史（临时对话）
tada chat --no-history
```

### 7.2 CLI 参数

```go
// cmd/tada/chat.go

flags:
--prompt, -p      # 指定 prompt 模板
--continue, -c    # 恢复对话
--list, -l        # 列出所有对话
--today           # 仅列出今天的对话
--date            # 列出指定日期的对话
--show, -s        # 显示对话详情
--delete, -d      # 删除对话
--name, -n        # 新对话名称
--no-history      # 不保存历史
--no-stream       # 禁用流式输出
--no-render       # 禁用 markdown 渲染
```

## 8. 用户交互示例

### 8.1 新对话

```
$ tada chat --prompt coder
📝 新对话 (coder)
💬 输入消息，/help 查看命令，/exit 退出

> 我如何在 Go 中解析 JSON？

🤠 思考中...
[流式显示原始文本]

[清屏后显示渲染后的 markdown，代码高亮、格式美观]

> /exit
📝 对话已保存
   ID: abc123-def456
   日期: 2026-02-22
   消息: 2 条
   恢复: tada chat --continue abc123-def456
```

### 8.2 恢复对话

```
$ tada chat --continue abc123-def456
📂 恢复对话: abc123-def456 (coder)
💬 最后更新: 2 小时前

[显示历史消息摘要]

> 继续
```

### 8.3 列出对话

```
$ tada chat --list
💬 对话历史:

今天 (2026-02-22):
  abc123-def456  [coder]    2 条消息  2 小时前
  def456-ghi789  [default]  5 条消息  1 小时前

昨天 (2026-02-21):
  xyz111-uvw222  [expert]   15 条消息
```

## 9. 测试策略

### 9.1 单元测试
- `Manager` 的创建、加载、保存、删除逻辑
- `PromptLoader` 的解析和验证
- `Storage` 的文件操作和日期路径处理
- `Renderer` 的 markdown 渲染

### 9.2 集成测试
- 完整对话流程（新建 → 消息 → 保存 → 恢复）
- Prompt 切换
- 流式输出
- Markdown 渲染
- 日期分组存储

### 9.3 E2E 测试
- 真实 AI API 调用（需要 `TADA_INTEGRATION_TEST=1`）

## 10. 依赖添加

```go
// go.mod 添加
require (
    github.com/charmbracelet/glamour v0.8.0  // Markdown 渲染
)
```

## 11. 实现优先级

### Phase 1: 基础对话（核心）
- conversation 包结构
- Manager、Storage 基础功能（含日期路径）
- chatCmd 重写
- AIProvider.ChatStream (OpenAI/GLM)

### Phase 2: Prompt 系统
- PromptLoader
- 配置文件扩展
- 默认 prompt 模板

### Phase 3: REPL 交互
- REPL 组件
- 对话恢复
- 命令系统

### Phase 4: Markdown 渲染
- Renderer 组件
- glamour 集成
- REPL 流式 + 渲染显示

### Phase 5: 完善功能
- --list (含日期过滤)
- --show, --delete
- 对话命名
- 错误处理和提示
- 交互优化
