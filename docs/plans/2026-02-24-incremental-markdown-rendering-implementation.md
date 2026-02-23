# Incremental Markdown Rendering Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement differential incremental markdown rendering for tada chat to eliminate visual flicker and provide real-time formatted output during streaming.

**Architecture:** Create a new `IncrementalRenderer` component that performs full markdown rendering on each chunk, diffs against previous output, and only redraws changed lines using precise ANSI cursor manipulation.

**Tech Stack:** Go 1.x, Glamour (markdown rendering), ANSI escape codes (terminal control)

---

## Task 1: Create IncrementalRenderer scaffold

**Files:**
- Create: `internal/conversation/incremental_renderer.go`
- Test: `internal/conversation/incremental_renderer_test.go`

**Step 1: Write failing test for NewIncrementalRenderer**

```go
// internal/conversation/incremental_renderer_test.go
package conversation

import (
	"testing"
)

func TestNewIncrementalRenderer(t *testing.T) {
	width := 80
	ir, err := NewIncrementalRenderer(width)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ir == nil {
		t.Fatal("Expected renderer to be created")
	}

	if ir.width != width {
		t.Errorf("Expected width %d, got %d", width, ir.width)
	}

	if !ir.isFirst {
		t.Error("Expected isFirst to be true for new renderer")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/conversation -run TestNewIncrementalRenderer -v`

Expected: `undefined: NewIncrementalRenderer`

**Step 3: Write minimal implementation**

```go
// internal/conversation/incremental_renderer.go
package conversation

import (
	"fmt"
)

// IncrementalRenderer 增量渲染器
type IncrementalRenderer struct {
	baseRenderer *Renderer // 基础 Glamour 渲染器
	width        int       // 终端宽度
	oldLines     []string  // 上次渲染的行
	lineCount    int       // 上次渲染的总行数
	isFirst      bool      // 是否首次渲染
}

// NewIncrementalRenderer 创建增量渲染器
func NewIncrementalRenderer(width int) (*IncrementalRenderer, error) {
	baseRenderer, err := NewRenderer(width)
	if err != nil {
		return nil, err
	}

	return &IncrementalRenderer{
		baseRenderer: baseRenderer,
		width:        width,
		oldLines:     nil,
		lineCount:    0,
		isFirst:      true,
	}, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/conversation -run TestNewIncrementalRenderer -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/conversation/incremental_renderer.go internal/conversation/incremental_renderer_test.go
git commit -m "feat: scaffold IncrementalRenderer with constructor"
```

---

## Task 2: Implement RenderIncremental - First Render

**Files:**
- Modify: `internal/conversation/incremental_renderer.go`
- Modify: `internal/conversation/incremental_renderer_test.go`

**Step 1: Write failing test for first render**

```go
func TestRenderIncremental_FirstRender(t *testing.T) {
	ir, _ := NewIncrementalRenderer(80)

	// First render should render without diff
	markdown := "# Hello\nWorld"
	err := ir.RenderIncremental(markdown)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ir.isFirst {
		t.Error("Expected isFirst to be false after first render")
	}

	if len(ir.oldLines) == 0 {
		t.Error("Expected oldLines to be populated after render")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/conversation -run TestRenderIncremental_FirstRender -v`

Expected: `undefined: IncrementalRenderer.RenderIncremental`

**Step 3: Write minimal implementation (first render only)**

```go
// RenderIncremental 增量渲染 markdown
func (ir *IncrementalRenderer) RenderIncremental(markdown string) error {
	// 使用基础渲染器渲染完整 markdown
	rendered, err := ir.baseRenderer.Render(markdown)
	if err != nil {
		return err
	}

	// 按行切分
	newLines := splitLines(rendered)

	if ir.isFirst {
		// 首次渲染：直接输出所有内容
		for _, line := range newLines {
			fmt.Println(line)
		}
		ir.isFirst = false
		ir.oldLines = newLines
		ir.lineCount = len(newLines)
		return nil
	}

	// Diff 逻辑在下一个任务实现
	return nil
}

// splitLines 按行切分字符串
func splitLines(s string) []string {
	lines := make([]string, 0)
	current := ""

	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}

	// 添加最后一行（可能没有换行符）
	if current != "" || len(lines) == 0 {
		lines = append(lines, current)
	}

	return lines
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/conversation -run TestRenderIncremental_FirstRender -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/conversation/incremental_renderer.go internal/conversation/incremental_renderer_test.go
git commit -m "feat: implement RenderIncremental first render logic"
```

---

## Task 3: Implement RenderIncremental - Diff Logic

**Files:**
- Modify: `internal/conversation/incremental_renderer.go`
- Modify: `internal/conversation/incremental_renderer_test.go`

**Step 1: Write failing test for diff rendering**

```go
func TestRenderIncremental_DiffRender(t *testing.T) {
	// 测试需要捕获 stdout，这里简化测试 diff 逻辑
	ir, _ := NewIncrementalRenderer(80)

	// First render
	ir.RenderIncremental("# Hello\n")

	// Store old lines
	oldLineCount := ir.lineCount
	oldLinesCopy := make([]string, len(ir.oldLines))
	copy(oldLinesCopy, ir.oldLines)

	// Second render with more content
	ir.RenderIncremental("# Hello\nWorld\n")

	// Verify state was updated
	if ir.lineCount <= oldLineCount {
		t.Errorf("Expected lineCount to increase, was %d now %d", oldLineCount, ir.lineCount)
	}

	if len(ir.oldLines) <= len(oldLinesCopy) {
		t.Error("Expected oldLines to grow")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/conversation -run TestRenderIncremental_DiffRender -v`

Expected: Test may pass but diff logic not implemented correctly

**Step 3: Implement diff logic**

```go
// RenderIncremental 增量渲染 markdown
func (ir *IncrementalRenderer) RenderIncremental(markdown string) error {
	// 使用基础渲染器渲染完整 markdown
	rendered, err := ir.baseRenderer.Render(markdown)
	if err != nil {
		return err
	}

	// 按行切分
	newLines := splitLines(rendered)

	if ir.isFirst {
		// 首次渲染：直接输出所有内容
		for _, line := range newLines {
			fmt.Println(line)
		}
		ir.isFirst = false
		ir.oldLines = newLines
		ir.lineCount = len(newLines)
		return nil
	}

	// Diff: 找到第一个不同的行
	diffIndex := findDiffIndex(ir.oldLines, newLines)

	if diffIndex == -1 {
		// 内容完全相同，不需要重绘
		return nil
	}

	// 计算需要向上移动的行数
	moveUp := ir.lineCount - diffIndex

	// 光标回退到差异行
	fmt.Printf("\033[%dA", moveUp)

	// 清除从光标到屏幕末尾的内容
	fmt.Print("\033[J")

	// 从差异行开始重绘
	for i := diffIndex; i < len(newLines); i++ {
		if i == len(newLines)-1 {
			// 最后一行不需要换行（避免额外空行）
			fmt.Print(newLines[i])
		} else {
			fmt.Println(newLines[i])
		}
	}

	// 更新状态
	ir.oldLines = newLines
	ir.lineCount = len(newLines)

	return nil
}

// findDiffIndex 找到两个切片的第一个差异索引
func findDiffIndex(oldLines, newLines []string) int {
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	for i := 0; i < maxLen; i++ {
		if i >= len(oldLines) || i >= len(newLines) {
			return i
		}
		if oldLines[i] != newLines[i] {
			return i
		}
	}

	return -1 // 完全相同
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/conversation -run TestRenderIncremental_DiffRender -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/conversation/incremental_renderer.go internal/conversation/incremental_renderer_test.go
git commit -m "feat: implement diff-based incremental rendering"
```

---

## Task 4: Implement Reset method

**Files:**
- Modify: `internal/conversation/incremental_renderer.go`
- Modify: `internal/conversation/incremental_renderer_test.go`

**Step 1: Write failing test**

```go
func TestIncrementalRenderer_Reset(t *testing.T) {
	ir, _ := NewIncrementalRenderer(80)

	// Do a render
	ir.RenderIncremental("# Hello\n")

	// Reset
	ir.Reset()

	if !ir.isFirst {
		t.Error("Expected isFirst to be true after reset")
	}

	if len(ir.oldLines) != 0 {
		t.Error("Expected oldLines to be empty after reset")
	}

	if ir.lineCount != 0 {
		t.Error("Expected lineCount to be 0 after reset")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/conversation -run TestIncrementalRenderer_Reset -v`

Expected: `undefined: IncrementalRenderer.Reset`

**Step 3: Implement Reset**

```go
// Reset 重置渲染器状态 (用于 resize 后)
func (ir *IncrementalRenderer) Reset() {
	ir.oldLines = nil
	ir.lineCount = 0
	ir.isFirst = true
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/conversation -run TestIncrementalRenderer_Reset -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/conversation/incremental_renderer.go internal/conversation/incremental_renderer_test.go
git commit -m "feat: add Reset method for resize handling"
```

---

## Task 5: Implement SetWidth method

**Files:**
- Modify: `internal/conversation/incremental_renderer.go`
- Modify: `internal/conversation/incremental_renderer_test.go`

**Step 1: Write failing test**

```go
func TestIncrementalRenderer_SetWidth(t *testing.T) {
	ir, _ := NewIncrementalRenderer(80)

	// Set new width
	err := ir.SetWidth(100)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ir.width != 100 {
		t.Errorf("Expected width 100, got %d", ir.width)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/conversation -run TestIncrementalRenderer_SetWidth -v`

Expected: `undefined: IncrementalRenderer.SetWidth`

**Step 3: Implement SetWidth**

```go
// SetWidth 更新终端宽度 (用于 resize)
// 宽度变化后需要重新创建基础渲染器
func (ir *IncrementalRenderer) SetWidth(width int) error {
	baseRenderer, err := NewRenderer(width)
	if err != nil {
		return err
	}

	ir.baseRenderer = baseRenderer
	ir.width = width
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/conversation -run TestIncrementalRenderer_SetWidth -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/conversation/incremental_renderer.go internal/conversation/incremental_renderer_test.go
git commit -m "feat: add SetWidth method for terminal resize"
```

---

## Task 6: Integrate IncrementalRenderer into REPL

**Files:**
- Modify: `internal/terminal/repl.go`

**Step 1: Add IncrementalRenderer field to REPL**

```go
type REPL struct {
	manager            *conversation.Manager
	conversation       *conversation.Conversation
	renderer           *conversation.Renderer
	incrementalRenderer *conversation.IncrementalRenderer // 新增
	stream             bool
	showThinking       bool
}
```

**Step 2: Add SetIncrementalRenderer method**

```go
// SetIncrementalRenderer 设置增量渲染器
func (r *REPL) SetIncrementalRenderer(ir *conversation.IncrementalRenderer) {
	r.incrementalRenderer = ir
}
```

**Step 3: Modify processStreamChat to use incremental rendering**

Find and modify the `processStreamChat` method:

```go
// processStreamChat 处理流式对话
func (r *REPL) processStreamChat(input string) error {
	if r.showThinking {
		fmt.Print("🤠 思考中...")
	}

	stream, err := r.manager.ChatStream(r.conversation.ID, input)
	if err != nil {
		// 出错时清除思考提示
		if r.showThinking {
			fmt.Print("\r\033[K")
		}
		return err
	}

	// 在开始流式输出前清除思考提示
	if r.showThinking {
		fmt.Print("\r\033[K")
	}

	var fullResponse strings.Builder

	for chunk := range stream {
		fullResponse.WriteString(chunk)

		// 使用增量渲染
		if r.incrementalRenderer != nil {
			if err := r.incrementalRenderer.RenderIncremental(fullResponse.String()); err != nil {
				// 渲染失败时降级到原始输出
				fmt.Print(chunk)
			}
		} else {
			// 降级到原始流式输出
			fmt.Print(chunk)
		}
	}

	// 打印换行符
	fmt.Println()

	return nil
}
```

**Step 4: Remove old clear-and-redraw logic**

The old code that cleared streaming output and re-rendered is no longer needed. It has been replaced above.

**Step 5: Commit**

```bash
git add internal/terminal/repl.go
git commit -m "feat: integrate IncrementalRenderer into REPL"
```

---

## Task 7: Wire up IncrementalRenderer in chat command

**Files:**
- Modify: `cmd/tada/chat.go` (or wherever chatCmd is defined)

**Step 1: Find the chat command initialization**

Run: `grep -n "NewREPL" cmd/tada/*.go`

**Step 2: Add IncrementalRenderer creation**

Based on findings, create the IncrementalRenderer after creating the REPL:

```go
// 在创建 REPL 后添加
repl := terminal.NewREPL(manager, conv, *stream)

// 创建增量渲染器
width := getTerminalWidth() // 需要实现这个函数
incrementalRenderer, err := conversation.NewIncrementalRenderer(width)
if err != nil {
	log.Printf("Warning: failed to create incremental renderer: %v", err)
} else {
	repl.SetIncrementalRenderer(incrementalRenderer)
}
```

**Step 3: Implement getTerminalWidth helper**

Add this helper function:

```go
import (
	"os"
	"golang.org/x/term"
)

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // 默认宽度
	}
	return width
}

// 检查 golang.org/x/term 是否已存在
// 如果没有，使用: go get golang.org/x/term
```

**Step 4: Run go mod tidy**

```bash
go mod tidy
```

**Step 5: Commit**

```bash
git add cmd/tada/chat.go go.mod go.sum
git commit -m "feat: wire up IncrementalRenderer in chat command"
```

---

## Task 8: Add fallback when --no-render flag is used

**Files:**
- Modify: `cmd/tada/chat.go`

**Step 1: Check existing --no-render logic**

Run: `grep -n "noRender\|no-render" cmd/tada/*.go`

**Step 2: Ensure IncrementalRenderer respects --no-render flag**

If `--no-render` flag is set, don't create IncrementalRenderer:

```go
// 只在启用渲染时创建增量渲染器
if !noRender {
	width := getTerminalWidth()
	incrementalRenderer, err := conversation.NewIncrementalRenderer(width)
	if err != nil {
		log.Printf("Warning: failed to create incremental renderer: %v", err)
	} else {
		repl.SetIncrementalRenderer(incrementalRenderer)
	}
}
```

**Step 3: Commit**

```bash
git add cmd/tada/chat.go
git commit -m "feat: respect --no-render flag for incremental rendering"
```

---

## Task 9: Add integration test

**Files:**
- Create: `tests/integration/incremental_rendering_test.go`

**Step 1: Write integration test**

```go
// tests/integration/incremental_rendering_test.go
// +build integration

package integration

import (
	"strings"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/conversation"
)

func TestIncrementalRendering_Integration(t *testing.T) {
	// 跳过如果不是集成测试
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ir, err := conversation.NewIncrementalRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	// 模拟流式输入
	chunks := []string{
		"# Hello\n",
		"## ",
		"World\n",
		"```",
		"go\n",
		"fmt.Println",
		"(\"hi\")\n",
		"```\n",
	}

	var full strings.Builder
	for _, chunk := range chunks {
		full.WriteString(chunk)
		err := ir.RenderIncremental(full.String())
		if err != nil {
			t.Errorf("RenderIncremental failed: %v", err)
		}
	}

	// 验证最终状态
	if ir.lineCount == 0 {
		t.Error("Expected non-zero line count")
	}
}
```

**Step 2: Run integration test**

```bash
TADA_INTEGRATION_TEST=1 go test ./tests/integration -run TestIncrementalRendering_Integration -v
```

**Step 3: Commit**

```bash
git add tests/integration/incremental_rendering_test.go
git commit -m "test: add integration test for incremental rendering"
```

---

## Task 10: Manual testing checklist

**Files:** N/A

**Step 1: Build the project**

```bash
go build -o tada cmd/tada/main.go
```

**Step 2: Test basic streaming**

```bash
./tada chat
# Enter: "写一个 Go hello world"
# Verify: No flicker, formatted output appears incrementally
```

**Step 3: Test code block streaming**

```bash
# In chat, enter: "写一个快速排序算法"
# Verify: Code block format appears during streaming
```

**Step 4: Test long content**

```bash
# In chat, enter: "详细介绍 Go 语言的历史" (expecting long response)
# Verify: Performance is acceptable
```

**Step 5: Test --no-render flag**

```bash
./tada chat --no-render
# Verify: Falls back to plain text streaming
```

**Step 6: Commit**

```bash
git add docs/plans/2026-02-24-incremental-markdown-rendering-implementation.md
git commit -m "docs: complete incremental rendering implementation plan"
```

---

## Testing Strategy Summary

1. **Unit Tests**: Each IncrementalRenderer method has dedicated tests
2. **Integration Test**: Full streaming workflow test
3. **Manual Testing**: Real-world usage scenarios

## Dependencies

- `github.com/charmbracelet/glamour` - Already in go.mod
- `golang.org/x/term` - For terminal width detection

## Notes

- The `splitLines` function preserves empty lines (unlike `strings.Split`)
- ANSI escape codes used: `\033[<n>A` (move up), `\033[J` (clear to end)
- First render bypasses diff logic for simplicity
- Empty responses are handled gracefully (lineCount may be 0)
