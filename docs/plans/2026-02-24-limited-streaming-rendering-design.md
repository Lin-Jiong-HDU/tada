# 有限流式输出渲染设计文档

**日期**: 2026-02-24
**状态**: 设计阶段

## 概述

为 tada chat 实现有限流式输出系统，解决当前终端自动换行导致原文清除不干净的问题。通过追踪显式换行符和终端宽度来计算行数，限制流式输出显示行数，超出后显示省略号。

## 问题陈述

### 当前实现的问题

当前 `processStreamChat` 的渲染流程：

1. 显示"思考中"提示
2. 接收流式 markdown chunks
3. **实时打印原始文本**（无格式）
4. 流结束后**清除所有原始输出**（使用 ANSI 转义码 `\033[%dA\033[J`）
5. **一次性渲染完整 markdown**

### 核心问题

```
lineCount := 1
for chunk := range stream {
    fmt.Print(chunk)
    lineCount += strings.Count(chunk, "\n")  // 只计数显式换行
}
fmt.Printf("\033[%dA\033[J", lineCount)  // 清除原文
```

| 问题 | 影响 |
|------|------|
| 自动换行未追踪 | 终端宽度限制导致长文本自动换行，但 `lineCount` 没有计入 |
| 清除不完整 | 追踪的行数少于实际视觉行数，导致部分原文残留 |
| 视觉混乱 | 渲染后的 markdown 与残留的原始文本混合 |

### 之前的尝试

已在 `docs/plans/2026-02-23-incremental-markdown-rendering-design.md` 中设计了增量渲染方案，通过实时渲染 markdown 来避免清除问题。本设计提供了一个更简单的替代方案。

## 设计方案

### 核心思路

```
流式输出时:
  1. 追踪显式 \n + 计算自动换行（基于终端宽度）
  2. 限制显示为固定行数（如 10 行）
  3. 超出后停止显示新内容，输出 "..."
  4. 流结束后使用追踪的行数清除原文
  5. 渲染完整 markdown
```

### 算法流程

```
收到 Chunk
    ↓
获取终端宽度
    ↓
计算该 chunk 会产生的行数（显式 + 自动换行）
    ↓
当前总行数 + chunk 行数 <= maxLines?
    ↓
  是 → 显示 chunk，更新行数
  否 → 已停止？→ 跳过
       未停止？→ 显示 "..."，标记已停止
    ↓
流结束后:
  使用追踪的总行数清除原文
  渲染完整 markdown
```

### 技术选择

| 决策点 | 选择 | 理由 |
|--------|------|------|
| 行数追踪 | 显式 `\n` + 终端宽度计算 | 尽可能准确反映实际视觉行数 |
| 超限后行为 | 显示 "..." 并停止更新 | 清晰提示有更多内容 |
| 省略号样式 | 简单 "..." | 无需额外样式，简洁 |
| 配置方式 | YAML 配置文件 | 与项目现有配置方式一致 |
| 默认行数 | 10 行 | 平衡阅读体验和屏幕占用 |

## 架构设计

### 新增配置

**`~/.tada/config.yaml`**:

```yaml
chat:
  streaming:
    max_display_lines: 10  # 流式输出最大显示行数，默认 10，设为 0 表示不限制
```

### 新组件: `LineTracker`

位置: `internal/terminal/tracker.go`

```go
// LineTracker 追踪流式输出的行数
type LineTracker struct {
    maxWidth    int     // 终端宽度
    currentPos  int     // 当前行内的字符位置
    lineCount   int     // 总行数（包括自动换行）
    maxLines    int     // 最大行数限制
    stopped     bool    // 是否已停止显示（超限后）
}

// NewLineTracker 创建行数追踪器
func NewLineTracker(maxLines int) (*LineTracker, error)

// Track 追踪文本，返回应该显示的文本和是否超限
// 返回值:
//   - displayText: 应该显示的文本（超限后为空）
//   - overflow: 是否超限（首次超限时返回 true）
func (t *LineTracker) Track(text string) (displayText string, overflow bool)

// LineCount 返回当前追踪的总行数
func (t *LineTracker) LineCount() int

// Reset 重置追踪器状态
func (t *LineTracker) Reset()
```

### 修改现有组件

#### `internal/terminal/repl.go`

```go
// REPL 结构体添加字段
type REPL struct {
    manager      *conversation.Manager
    conversation *conversation.Conversation
    renderer     *conversation.Renderer
    stream       bool
    showThinking bool
    maxDisplayLines int  // 新增：最大显示行数
}

// NewREPL 构造函数添加参数
func NewREPL(manager *conversation.Manager, conv *conversation.Conversation,
    stream bool, maxDisplayLines int) *REPL {
    // ...
}

// processStreamChat 修改
func (r *REPL) processStreamChat(input string) error {
    // ... 前面的思考提示逻辑不变 ...

    var fullResponse strings.Builder
    tracker, err := NewLineTracker(r.maxDisplayLines)
    if err != nil {
        // 降级到原始行为
        return r.processStreamChatFallback(input)
    }

    for chunk := range stream {
        fullResponse.WriteString(chunk)

        // 使用追踪器处理显示
        displayText, overflow := tracker.Track(chunk)
        if displayText != "" {
            fmt.Print(displayText)
        }
        if overflow && !tracker.Stopped() {
            fmt.Print("...")
        }
    }

    // 清除流式输出的原文
    fmt.Printf("\033[%dA\033[J", tracker.LineCount())

    // 渲染美化版本
    fmt.Print("\n🤖\n")
    if r.renderer != nil {
        rendered, _ := r.renderer.Render(fullResponse.String())
        fmt.Print(rendered)
    } else {
        fmt.Println(fullResponse.String())
    }

    return nil
}
```

#### `cmd/tada/chat.go`

添加配置读取和传递：

```go
// 从配置读取 max_display_lines
maxDisplayLines := 10 // 默认值
if cfg.Chat.Streaming.MaxDisplayLines > 0 {
    maxDisplayLines = cfg.Chat.Streaming.MaxDisplayLines
}

repl := terminal.NewREPL(manager, conv, *stream, maxDisplayLines)
```

## 数据流图

```
┌─────────────────────────────────────────────────────────────┐
│                        REPL                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              processStreamChat                       │    │
│  │                                                       │    │
│  │   tracker := NewLineTracker(maxLines)                │    │
│  │   for chunk := range stream {                        │    │
│  │       fullResponse += chunk                          │    │
│  │       displayText, overflow := tracker.Track(chunk)  │    │
│  │       if displayText != "" {                         │    │
│  │           fmt.Print(displayText)  // 限制行数显示    │    │
│  │       }                                              │    │
│  │       if overflow {                                  │    │
│  │           fmt.Print("...")                           │    │
│  │       }                                              │    │
│  │   }                                                  │    │
│  │   fmt.Printf("\033[%dA\033[J", tracker.LineCount()) │    │
│  │   renderer.Render(fullResponse)  // 最终渲染         │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                     LineTracker                              │
│                                                              │
│   Track(chunk) {                                             │
│       for each char in chunk {                               │
│           if char == '\n' {                                  │
│               lineCount++                                    │
│               currentPos = 0                                 │
│           } else if currentPos >= maxWidth {                 │
│               lineCount++  // 自动换行                       │
│               currentPos = 1  // 当前字符占用新行            │
│           } else {                                           │
│               currentPos++                                   │
│           }                                                  │
│       }                                                      │
│       if lineCount > maxLines {                              │
│           return "", true  // 停止显示                       │
│       }                                                      │
│       return chunk, false                                    │
│   }                                                          │
└─────────────────────────────────────────────────────────────┘
```

## 行数计算算法

### 核心逻辑

```go
func (t *LineTracker) Track(text string) (displayText string, overflow bool) {
    if t.stopped {
        return "", false
    }

    var result strings.Builder
    for _, r := range text {
        if r == '\n' {
            t.lineCount++
            t.currentPos = 0
        } else {
            if t.currentPos >= t.maxWidth {
                // 需要自动换行
                t.lineCount++
                t.currentPos = 0
            }
            t.currentPos++
        }

        if t.lineCount > t.maxLines {
            t.stopped = true
            if result.Len() > 0 {
                return result.String(), true
            }
            return "", true
        }

        result.WriteRune(r)
    }

    return result.String(), false
}
```

### 处理特殊情况

| 场景 | 处理方式 |
|------|----------|
| Tab 字符 | 按 8 字符计算（或配置 tab 宽度） |
| ANSI 转义序列 | 不计入宽度（需要检测和跳过） |
| 宽 Unicode 字符 | 使用 `unicode/runewidth.RuneWidth()` |
| 终端宽度获取失败 | 回退到只计数 `\n`，设置合理的默认宽度 |

## 配置结构

### Config 定义

```go
// internal/config/config.go
type Config struct {
    AI       AIConfig       `yaml:"ai"`
    Security SecurityConfig `yaml:"security"`
    Chat     ChatConfig     `yaml:"chat"`  // 新增
}

type ChatConfig struct {
    Streaming StreamingConfig `yaml:"streaming"`
}

type StreamingConfig struct {
    MaxDisplayLines int `yaml:"max_display_lines"`
}
```

### 默认值

```go
var DefaultConfig = Config{
    // ...
    Chat: ChatConfig{
        Streaming: StreamingConfig{
            MaxDisplayLines: 10,
        },
    },
}
```

## 边界情况处理

| 场景 | 处理方式 |
|------|----------|
| 终端宽度获取失败 | 回退到仅计数 `\n`，使用默认宽度 80 |
| 终端宽度变化 | 使用初始宽度，不动态调整 |
| maxLines = 0 | 表示不限制，完全流式输出 |
| 空响应 | 不渲染，lineCount 为 0 |
| 纯 ANSI 转义序列 | 检测并跳过，不计入宽度 |
| 超长单行 | 计算自动换行，正确计入行数 |

## 测试策略

### 单元测试

**`internal/terminal/tracker_test.go`**:

```go
func TestLineTracker_NewTracker(t *testing.T)
func TestLineTrack_SimpleText(t *testing.T)
func TestLineTrack_WithNewlines(t *testing.T)
func TestLineTrack_WithWrap(t *testing.T)
func TestLineTrack_Overflow(t *testing.T)
func TestLineTrack_Reset(t *testing.T)
func TestLineTrack_WideChars(t *testing.T)
func TestLineTrack_ANSIEscapeCodes(t *testing.T)
```

### 集成测试

**`tests/integration/limited_streaming_test.go`**:

```go
func TestLimitedStreaming_FullFlow(t *testing.T)
func TestLimitedStreaming_OverflowBehavior(t *testing.T)
func TestLimitedStreaming_ConfigureMaxLines(t *testing.T)
```

### 手动测试场景

1. **短文本** - 未超限，正常显示和清除
2. **长文本** - 超限后显示 "..."
3. **代码块** - 验证格式正确
4. **窄终端** - 测试自动换行计算
5. **配置为 0** - 验证不限制模式
6. **配置为负数** - 使用默认值

## 性能考虑

| 操作 | 复杂度 | 说明 |
|------|--------|------|
| 字符宽度计算 | O(1) | 使用查表法 |
| 行数追踪 | O(n) | n = chunk 字符数 |
| 宽字符处理 | O(1) | `unicode/runewidth` |
| ANSI 检测 | O(n) | 需要解析转义序列 |

**优化点**:
- ANSI 转义序列检测可以缓存
- 对于纯 ASCII 文本可快速路径处理

## 依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| `golang.org/x/term` | - | 获取终端宽度 |
| `github.com/mattn/go-runewidth` | - | 字符宽度计算（更精确） |

## 与增量渲染方案的对比

| 特性 | 有限流式输出 | 增量渲染 |
|------|-------------|----------|
| 实现复杂度 | 中等 | 高 |
| 流式过程显示 | 纯文本（限制行数） | 格式化 markdown |
| 视觉效果 | 简单 | 更佳 |
| 清除问题 | 通过限制行数缓解 | 通过增量渲染避免 |
| 性能 | 较低 | 较高（每个 chunk 都渲染） |
| 回退选项 | 容易 | 需要额外逻辑 |

## 后续改进

1. **可配置省略号样式** - 允许用户自定义省略号显示
2. **动态终端宽度** - 检测并响应窗口大小变化
3. **智能行数** - 根据终端高度动态调整 maxLines
4. **进度指示** - 超限时显示进度条

## 实施计划

详见: `docs/plans/2026-02-24-limited-streaming-rendering-implementation.md` (待创建)
