# tada 设计文档

> **"Tada! And it's done."**
>
> 创建日期: 2025-02-18

---

## 1. 项目概述

`tada` 是一个用 Go 编写的轻量级终端智能助手。它能理解用户的自然语言意图并自动执行命令，让用户从繁琐的 CLI 语法中解脱出来。

**核心特性：**
- 直接 CLI 调用：`tada <自然语言请求>`
- 按需 TUI：仅在需要授权/查询时进入交互界面
- 插件系统：支持注入式和工具式两种扩展方式
- 轻量化：纯文件系统存储，无重型依赖
- AI 抽象：易于切换云端/本地模型

---

## 2. 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                           用户交互层                              │
│  ┌──────────────┐  ┌────────────────────────────────────────┐   │
│  │   CLI 入口   │  │            TUI 界面                     │   │
│  │  (同步命令)  │  │  (确认/查询 │ 后台任务管理 │ 配置等)     │   │
│  └──────┬───────┘  └────────────────┬───────────────────────┘   │
└─────────┼───────────────────────────┼───────────────────────────┘
          │                           │
          ▼                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                          核心引擎层                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  会话管理器  │  │  安全控制器  │  │    任务执行器        │  │
│  │ (上下文/历史)│  │ (权限/沙箱)  │  │  (命令运行/流式)     │  │
│  └──────────────┘  └──────────────┘  └──────────┬───────────┘  │
│                                                  │               │
│  ┌──────────────────────────────────────────────┴─────────┐    │
│  │                    意图解析引擎                          │    │
│  │  (自然语言 → 结构化指令 + LLM 分析输出)                  │    │
│  └───────────────────────────┬──────────────────────────────┘    │
└──────────────────────────────┼──────────────────────────────────┘
                               │
          ┌────────────────────┼────────────────────┐
          ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│   插件系统      │  │   AI 后端       │  │   存储层        │
│ (注入+工具定义) │  │ (抽象接口)      │  │ (文件系统)      │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

---

## 3. 核心组件

### 3.1 AI Provider 接口

```go
// 核心 AI 抽象，便于扩展
type AIProvider interface {
    // 解析意图，返回结构化指令
    ParseIntent(ctx context.Context, input string, plugins []Plugin) (*Intent, error)

    // 分析命令输出，生成自然语言反馈
    AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error)

    // 普通对话（支持上下文，用于聊天模式）
    Chat(ctx context.Context, messages []Message) (string, error)

    // 流式对话（支持 TUI 聊天模式）
    ChatStream(ctx context.Context, messages []Message) (<-chan string, error)
}

type Message struct {
    Role    string  // "system" | "user" | "assistant"
    Content string
}

type Intent struct {
    Commands     []Command      // 要执行的命令列表
    Reason       string         // LLM 的推理过程
    NeedsConfirm bool           // 是否需要用户确认
}
```

### 3.2 插件系统

```go
// 插件接口（统一抽象）
type Plugin interface {
    Name() string
    Type() PluginType  // "injection" 或 "tool"

    // 注入式：返回额外提示词
    Prompt() string

    // 工具式：返回可用工具定义
    Tools() []Tool
}

type Tool struct {
    Name            string
    Description     string
    CommandTemplate string   // 模板，如 "task add {{.description}}"
    Parameters      []Parameter
}
```

### 3.3 安全控制器

```go
// 安全策略配置
type SecurityPolicy struct {
    CommandLevel    ConfirmLevel   // always | dangerous | never
    RestrictedPaths []string       // 禁止访问的路径
    ReadOnlyPaths   []string       // 只读路径
    AllowShell      bool           // 是否允许 shell 命令
}

type ConfirmLevel string
const (
    ConfirmAlways     ConfirmLevel = "always"
    ConfirmDangerous  ConfirmLevel = "dangerous"
    ConfirmNever      ConfirmLevel = "never"
)
```

---

## 4. 数据流

### 4.1 同步命令执行流程

```
用户输入 → tada 新建docs文件夹
    │
    ▼
CLI 解析参数 → 检测同步模式
    │
    ▼
加载会话历史 → 从 ~/.tada/sessions/current.json
    │
    ▼
加载插件 → 扫描 ~/.tada/plugins/
    │
    ▼
AI 解析意图 → ParseIntent()
    │
    ▼
安全检查 → 是否需要确认
    │
    ├─ 需要确认 → TUI 确认 ─┐
    │                      │
    └─ 无需确认 ────────────┤
                           │
                           ▼
                    执行命令 → 捕获输出
                           │
                           ▼
                    AI 分析结果 → AnalyzeOutput()
                           │
                           ▼
                    显示结果 → 固定行数 + LLM 分析
                           │
                           ▼
                    保存会话 → 更新 current.json
```

### 4.2 异步命令执行流程

```
用户输入 → tada 编译项目 &
    │
    ▼
CLI 解析 → 检测 & 后缀，异步模式
    │
    ▼
创建后台任务 → 生成 TaskID
    │
    ▼
立即返回 → "[tada] Task abc123 running"

    (后台 goroutine 继续)
    │
    ▼
解析 → 执行 → 分析
    │
    ▼
更新任务状态 → ~/.tada/tasks/abc123.json
```

---

## 5. 项目目录结构

```
tada/
├── cmd/
│   └── tada/
│       └── main.go                 # CLI 入口
│
├── internal/
│   ├── core/                       # 核心引擎
│   │   ├── engine.go               # 主引擎编排
│   │   ├── session.go              # 会话管理
│   │   ├── security.go             # 安全控制
│   │   └── executor.go             # 命令执行器
│   │
│   ├── ai/                         # AI 后端
│   │   ├── provider.go             # AIProvider 接口定义
│   │   ├── openai/
│   │   │   └── openai.go           # OpenAI 实现
│   │   └── claude/
│   │       └── claude.go           # Claude 实现（预留）
│   │
│   ├── plugin/                     # 插件系统
│   │   ├── loader.go               # 插件加载器
│   │   ├── injection.go            # 注入式插件
│   │   └── tool.go                 # 工具式插件
│   │
│   ├── ui/                         # TUI 界面
│   │   ├── tui.go                  # Bubble Tea 主程序
│   │   ├── components/             # TUI 组件
│   │   │   ├── confirm.go          # 确认对话框
│   │   │   ├── tasks.go            # 任务列表
│   │   │   └── chat.go             # 聊天界面
│   │   └── styles.go               # 样式定义
│   │
│   └── storage/                    # 存储层
│       ├── config.go               # 配置读写
│       ├── session.go              # 会话存储
│       └── task.go                 # 任务存储
│
├── docs/
│   └── plans/
│       └── 2025-02-18-tada-design.md
│
├── .tada/                          # 用户数据目录（运行时生成）
│   ├── config.yaml
│   ├── plugins/
│   ├── sessions/
│   └── tasks/
│
├── go.mod
└── README.md
```

---

## 6. 配置设计

### 6.1 主配置文件 (~/.tada/config.yaml)

```yaml
# AI 配置
ai:
  provider: openai
  api_key: sk-xxx
  model: gpt-4o
  base_url: https://api.openai.com/v1
  timeout: 30s
  max_tokens: 4096

# 安全配置
security:
  command_level: dangerous
  allow_shell: true
  restricted_paths:
    - /etc
    - /usr/bin
  readonly_paths:
    - ~/.ssh
  allow_terminal_takeover: true

# 会话配置
session:
  persist_history: true
  max_history: 100
  context_window: 10

# 输出配置
output:
  max_lines: 20
  always_analyze: true

# UI 配置
ui:
  language: zh
  theme: default
```

### 6.2 插件示例

**注入式插件** (~/.tada/plugins/task.md):
```markdown
# Taskwarrior 任务管理

Taskwarrior 是一个强大的命令行任务管理工具。

常用命令：
- `task add <description>` - 添加任务
- `task list` - 列出所有任务
- `task <id> done` - 完成任务

用户常用这个工具管理日常任务。
```

**工具式插件** (~/.tada/plugins/docker.yaml):
```yaml
name: docker
description: Docker 容器管理工具
tools:
  - name: docker_ps
    description: "列出运行中的容器"
    command_template: "docker ps"
    parameters: []

  - name: docker_run
    description: "运行一个容器"
    command_template: "docker run -d --name {{.name}} {{.image}}"
    parameters:
      - name: name
        description: 容器名称
        required: true
      - name: image
        description: 镜像名称
        required: true
```

---

## 7. 错误处理

### 7.1 错误分类

```go
type ErrorType string

const (
    ErrTypeAI        ErrorType = "ai"
    ErrTypeCommand   ErrorType = "command"
    ErrTypeSecurity  ErrorType = "security"
    ErrTypeConfig    ErrorType = "config"
    ErrTypePlugin    ErrorType = "plugin"
)

type TadaError struct {
    Type    ErrorType
    Message string
    Cause   error
    Hint    string
}
```

### 7.2 错误处理策略

| 场景 | 处理方式 |
|------|----------|
| AI API 调用失败 | 显示错误，询问是否重试 |
| 命令执行失败 | 显示错误 + 输出，询问是否继续 |
| 安全策略违规 | 明确拒绝，说明原因 |
| 配置文件错误 | 使用默认值 + 警告 |
| 插件加载失败 | 跳过该插件 + 警告 |

---

## 8. 测试策略

### 8.1 测试分层

```
tests/
├── unit/              # 单元测试
│   ├── ai/
│   ├── plugin/
│   └── storage/
│
├── integration/       # 集成测试
│   ├── engine_test.go
│   └── security_test.go
│
└── e2e/              # 端到端测试
    └── scenarios/
```

### 8.2 关键测试场景

- [ ] AI 解析意图 → 生成正确命令
- [ ] 安全策略正确拦截危险命令
- [ ] 异步任务状态正确更新
- [ ] 插件热加载工作正常
- [ ] 会话历史正确持久化
- [ ] TUI 各组件交互正常
- [ ] 并发请求处理

---

## 9. 实现阶段

### Phase 1: 核心基础 (MVP)
- CLI 入口 + 参数解析
- AI Provider 接口 + OpenAI 实现
- 简单意图解析 + 命令执行
- 基础配置管理

### Phase 2: 插件与安全
- 插件系统（注入式 + 工具式）
- 安全控制器
- 会话历史管理

### Phase 3: TUI 与异步
- Bubble Tea TUI
- 后台任务支持
- 终端接管功能

### Phase 4: 完善与优化
- 多 AI Provider 支持
- 国际化框架
- 更丰富的插件生态

---

## 10. 技术栈

- **语言**: Go 1.25.7
- **TUI 框架**: Bubble Tea (charmbracelet)
- **存储**: 文件系统 (YAML/TOML + JSON)
- **AI**: OpenAI API (可扩展)
- **配置**: Viper
- **CLI 框架**: Cobra (可选)
