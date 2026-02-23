# 增量 Markdown 渲染设计文档

**日期**: 2026-02-23
**状态**: 设计阶段

## 概述

为 tada chat 实现差异增量渲染系统，解决当前渲染方式存在的视觉闪烁、缺乏实时格式化和性能问题。

## 问题陈述

### 当前实现

当前 `processStreamChat` 的渲染流程：

1. 显示"思考中"提示
2. 接收流式 markdown chunks
3. **实时打印原始文本**（无格式）
4. 流结束后**清除所有原始输出**（使用 ANSI 转义码）
5. **一次性渲染完整 markdown**（Glamour）

### 存在的问题

| 问题 | 影响 |
|------|------|
| 视觉闪烁 | 清除再重绘产生明显的闪烁效果 |
| 无实时格式 | 用户在流式过程中看不到代码块、粗体等格式 |
| 性能问题 | 大文档一次性渲染耗时较长 |

## 设计方案

### 核心算法

```
收到 Chunk
    ↓
全量渲染 fullResponse (Glamour)
    ↓
按 \n 切分为 newLines
    ↓
Diff: 找到 newLines 与 oldLines 的第一个差异行索引 i
    ↓
光标回退: \033[<moveUp>A (moveUp = len(oldLines) - i)
    ↓
清除: \033[J (从光标到屏幕末尾)
    ↓
重绘: 打印 newLines[i:]
    ↓
更新: oldLines = newLines
```

### 技术选择

| 决策点 | 选择 | 理由 |
|--------|------|------|
| 渲染策略 | Raw incremental | 实现简单，Glamour 容错性好 |
| 光标处理 | Diff + 精确回退 | 只重绘变化部分，最小化屏幕更新 |
| 宽度处理 | 启动固定，Resize 清屏 | 简单可靠 |
| 错误处理 | 无需特殊处理 | Glamour 对不完整 markdown 不会报错 |
| 性能策略 | 实时渲染 | 不做节流，最快响应 |

## 架构设计

### 新组件: `IncrementalRenderer`

位置: `internal/conversation/incremental_renderer.go`

```go
// IncrementalRenderer 增量渲染器
type IncrementalRenderer struct {
    baseRenderer *Renderer       // 基础 Glamour 渲染器
    width        int             // 终端宽度
    oldLines     []string        // 上次渲染的行
    lineCount    int             // 上次渲染的总行数
    isFirst      bool            // 是否首次渲染
}

// NewIncrementalRenderer 创建增量渲染器
func NewIncrementalRenderer(width int) (*IncrementalRenderer, error)

// RenderIncremental 增量渲染 markdown
// 每次调用时:
// 1. 渲染完整 markdown
// 2. 与上次结果做 diff
// 3. 只重绘变化的部分
func (ir *IncrementalRenderer) RenderIncremental(markdown string) error

// Reset 重置渲染器状态 (用于 resize 后)
func (ir *IncrementalRenderer) Reset()

// SetWidth 更新终端宽度 (用于 resize)
func (ir *IncrementalRenderer) SetWidth(width int) error
```

### 修改现有组件

#### `internal/terminal/repl.go`

```go
type REPL struct {
    // ...
    incrementalRenderer *conversation.IncrementalRenderer  // 新增
    // ...
}

func (r *REPL) processStreamChat(input string) error {
    // ... 获取 stream

    var fullResponse strings.Builder

    for chunk := range stream {
        fullResponse.WriteString(chunk)

        // 替换原来的 fmt.Print(chunk)
        // 使用增量渲染
        if r.incrementalRenderer != nil {
            r.incrementalRenderer.RenderIncremental(fullResponse.String())
        } else {
            // 降级到原始流式输出
            fmt.Print(chunk)
        }
    }

    // 不再需要清除和重绘
    // 流式过程中已经渲染完成

    return nil
}
```

## 数据流图

```
┌─────────────────────────────────────────────────────────────┐
│                        REPL                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              processStreamChat                       │    │
│  │                                                       │    │
│  │   for chunk := range stream {                        │    │
│  │       fullResponse += chunk                          │    │
│  │       ↓                                              │    │
│  │   incrementalRenderer.RenderIncremental(fullResponse) │    │
│  │   }                                                  │    │
│  └─────────────────────────────────────────────────────┘    │
│                           ↓                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │         IncrementalRenderer                          │    │
│  │                                                       │    │
│  │   1. glamour.Render(fullResponse)                    │    │
│  │   2. Split by \n → newLines                         │    │
│  │   3. Diff(newLines, oldLines) → i                   │    │
│  │   4. fmt.Printf("\033[%dA\033[J", moveUp)           │    │
│  │   5. Print(newLines[i:])                            │    │
│  │   6. oldLines = newLines                            │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                              ↓
                        终端输出 (无闪烁)
```

## 边界情况处理

| 场景 | 处理方式 |
|------|----------|
| 终端宽度变化 | 检测到 resize 时重置渲染器，从头重绘 |
| Glamour 渲染失败 | 不需要处理，测试表明 Glamour 容错性好 |
| 首次渲染 | 直接输出，不做 diff |
| 用户流式期间输入 | REPL 流式期间阻塞输入 |
| 空响应 | 不渲染，保持空 |

## 测试策略

### 单元测试

- `TestIncrementalRenderer_NewRenderer` - 创建渲染器
- `TestIncrementalRenderer_FirstRender` - 首次渲染不 diff
- `TestIncrementalRenderer_DiffRender` - 差异渲染逻辑
- `TestIncrementalRenderer_Reset` - 重置状态

### 集成测试

- `TestREPL_StreamWithIncrementalRender` - 完整流式流程
- `TestREPL_RenderFallback` - 降级逻辑

### 手动测试场景

1. **基础流式渲染** - 观察无闪烁、实时格式
2. **代码块流式** - 验证不完整代码块的显示
3. **长文本** - 测试性能
4. **窗口 resize** - 验证重绘逻辑
5. **渲染器降级** - 禁用增量渲染时的行为

## 性能考虑

| 操作 | 复杂度 | 说明 |
|------|--------|------|
| Glamour 渲染 | O(n) | n = 文本长度 |
| Diff | O(m) | m = 行数 |
| 光标移动 | O(1) | 固定 ANSI 序列 |
| 屏幕输出 | O(k) | k = 变化行数 |

**优化点**: 只重绘变化部分，而非整屏

## 依赖

- `github.com/charmbracelet/glamour` - Markdown 渲染（已存在）
- 标准库 `strings` - 字符串操作

## 后续改进

1. **节流选项** - 可配置的渲染节流策略
2. **智能 Diff** - 基于单词而非行的 diff（更精细但更复杂）
3. **动画效果** - 流式结束后的淡入效果
4. **性能监控** - 渲染耗时统计

## 实施计划

详见: `docs/plans/2026-02-23-incremental-markdown-rendering-implementation.md`
