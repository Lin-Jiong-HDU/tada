# Limited Streaming Rendering Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement limited streaming output for tada chat to solve terminal line wrap clearing issues by tracking explicit newlines and automatic line wraps based on terminal width.

**Architecture:** Create a new `LineTracker` component that tracks line count during streaming, limits displayed lines to a configurable maximum, shows "..." when overflow occurs, and uses the tracked line count to clear original text before rendering the final markdown.

**Tech Stack:** Go 1.x, golang.org/x/term (terminal width), mattn/go-runewidth (character width calculation)

---

## Task 1: Add go-runewidth dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add the dependency**

Run: `go get github.com/mattn/go-runewidth@latest`

Expected: Dependency added to go.mod and go.sum

**Step 2: Run go mod tidy**

Run: `go mod tidy`

Expected: Dependencies cleaned up

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add mattn/go-runewidth for character width calculation"
```

---

## Task 2: Add streaming configuration to config structure

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Read the current config structure**

Run: `cat internal/config/config.go`

Expected: See current Config struct definition

**Step 2: Add Chat and Streaming config types**

Find the `Config` struct and add after `SecurityConfig`:

```go
// ChatConfig chat 配置
type ChatConfig struct {
	Streaming StreamingConfig `yaml:"streaming"`
}

// StreamingConfig 流式输出配置
type StreamingConfig struct {
	// MaxDisplayLines 流式输出最大显示行数，0 表示不限制
	MaxDisplayLines int `yaml:"max_display_lines"`
}
```

**Step 3: Add Chat field to Config struct**

Add to `Config` struct:

```go
type Config struct {
	AI       AIConfig       `yaml:"ai"`
	Security SecurityConfig `yaml:"security"`
	Chat     ChatConfig     `yaml:"chat"` // 新增

	// ... 其他字段
}
```

**Step 4: Update DefaultConfig**

Find `DefaultConfig` variable and add:

```go
var DefaultConfig = Config{
	AI: AIConfig{
		// ... 现有字段
	},
	Security: SecurityConfig{
		// ... 现有字段
	},
	Chat: ChatConfig{ // 新增
		Streaming: StreamingConfig{
			MaxDisplayLines: 10,
		},
	},
	// ... 其他字段
}
```

**Step 5: Verify code compiles**

Run: `go build ./internal/config`

Expected: No errors

**Step 6: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add streaming config to Config struct"
```

---

## Task 3: Create LineTracker scaffold

**Files:**
- Create: `internal/terminal/tracker.go`
- Create: `internal/terminal/tracker_test.go`

**Step 1: Write failing test for NewLineTracker**

```go
// internal/terminal/tracker_test.go
package terminal

import (
	"testing"
)

func TestNewLineTracker(t *testing.T) {
	maxLines := 10
	tracker, err := NewLineTracker(maxLines)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if tracker == nil {
		t.Fatal("Expected tracker to be created")
	}

	if tracker.maxLines != maxLines {
		t.Errorf("Expected maxLines %d, got %d", maxLines, tracker.maxLines)
	}

	if tracker.stopped {
		t.Error("Expected stopped to be false for new tracker")
	}

	if tracker.lineCount != 1 {
		t.Errorf("Expected initial lineCount 1, got %d", tracker.lineCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/terminal -run TestNewLineTracker -v`

Expected: `undefined: NewLineTracker`

**Step 3: Write minimal implementation**

```go
// internal/terminal/tracker.go
package terminal

import (
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// LineTracker 追踪流式输出的行数
type LineTracker struct {
	maxWidth    int     // 终端宽度
	currentPos  int     // 当前行内的字符位置
	lineCount   int     // 总行数（包括自动换行）
	maxLines    int     // 最大行数限制
	stopped     bool    // 是否已停止显示（超限后）
}

// NewLineTracker 创建行数追踪器
func NewLineTracker(maxLines int) (*LineTracker, error) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80 // 默认宽度
	}

	return &LineTracker{
		maxWidth:   width,
		currentPos: 0,
		lineCount:  1, // 从第1行开始
		maxLines:   maxLines,
		stopped:    false,
	}, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/terminal -run TestNewLineTracker -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/terminal/tracker.go internal/terminal/tracker_test.go
git commit -m "feat: scaffold LineTracker with constructor"
```

---

## Task 4: Implement Track - Simple text without newlines

**Files:**
- Modify: `internal/terminal/tracker.go`
- Modify: `internal/terminal/tracker_test.go`

**Step 1: Write failing test for simple text**

```go
func TestLineTracker_SimpleText(t *testing.T) {
	tracker, _ := NewLineTracker(10)

	// Track simple text without newlines
	display, overflow := tracker.Track("Hello")

	if overflow {
		t.Error("Expected no overflow for short text")
	}

	if display != "Hello" {
		t.Errorf("Expected display 'Hello', got '%s'", display)
	}

	if tracker.lineCount != 1 {
		t.Errorf("Expected lineCount 1, got %d", tracker.lineCount)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/terminal -run TestLineTracker_SimpleText -v`

Expected: `undefined: LineTracker.Track` or test fails

**Step 3: Implement Track method**

Add to `LineTracker`:

```go
// Track 追踪文本，返回应该显示的文本和是否超限
func (t *LineTracker) Track(text string) (displayText string, overflow bool) {
	if t.stopped {
		return "", false
	}

	var result strings.Builder

	for _, r := range text {
		charWidth := runewidth.RuneWidth(r)

		// 处理换行符
		if r == '\n' {
			t.lineCount++
			t.currentPos = 0
			result.WriteRune(r)
			if t.lineCount > t.maxLines {
				t.stopped = true
				return result.String(), true
			}
			continue
		}

		// 检查是否需要自动换行
		if t.currentPos+charWidth > t.maxWidth {
			t.lineCount++
			t.currentPos = 0
			if t.lineCount > t.maxLines {
				t.stopped = true
				return result.String(), true
			}
		}

		t.currentPos += charWidth
		result.WriteRune(r)
	}

	return result.String(), false
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/terminal -run TestLineTracker_SimpleText -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/terminal/tracker.go internal/terminal/tracker_test.go
git commit -m "feat: implement Track for simple text"
```

---

## Task 5: Implement Track - With newlines

**Files:**
- Modify: `internal/terminal/tracker_test.go`
- Modify: `internal/terminal/tracker.go`

**Step 1: Write failing test for text with newlines**

```go
func TestLineTracker_WithNewlines(t *testing.T) {
	tracker, _ := NewLineTracker(10)

	display, overflow := tracker.Track("Line 1\nLine 2\nLine 3")

	if overflow {
		t.Error("Expected no overflow for 3 lines")
	}

	if display != "Line 1\nLine 2\nLine 3" {
		t.Errorf("Unexpected display: '%s'", display)
	}

	if tracker.lineCount != 4 { // 1 + 3 newlines = 4 lines
		t.Errorf("Expected lineCount 4, got %d", tracker.lineCount)
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./internal/terminal -run TestLineTracker_WithNewlines -v`

Expected: `PASS` (implementation should already handle this)

**Step 3: Add more newline edge case tests**

```go
func TestLineTracker_TrailingNewline(t *testing.T) {
	tracker, _ := NewLineTracker(10)

	display, overflow := tracker.Track("Hello\n")

	if overflow {
		t.Error("Expected no overflow")
	}

	if tracker.lineCount != 2 {
		t.Errorf("Expected lineCount 2, got %d", tracker.lineCount)
	}
}

func TestLineTracker_EmptyString(t *testing.T) {
	tracker, _ := NewLineTracker(10)

	display, overflow := tracker.Track("")

	if overflow {
		t.Error("Expected no overflow for empty string")
	}

	if display != "" {
		t.Errorf("Expected empty display, got '%s'", display)
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/terminal -run TestLineTracker_ -v`

Expected: All `PASS`

**Step 5: Commit**

```bash
git add internal/terminal/tracker_test.go
git commit -m "test: add newline edge case tests"
```

---

## Task 6: Implement Track - With overflow

**Files:**
- Modify: `internal/terminal/tracker_test.go`
- Modify: `internal/terminal/tracker.go`

**Step 1: Write failing test for overflow behavior**

```go
func TestLineTracker_Overflow(t *testing.T) {
	// Set max to 2 lines
	tracker, _ := NewLineTracker(2)

	display1, overflow1 := tracker.Track("Line 1\n")

	if overflow1 {
		t.Error("Expected no overflow on first line")
	}

	// Second chunk causes overflow
	display2, overflow2 := tracker.Track("Line 2\nLine 3\n")

	if !overflow2 {
		t.Error("Expected overflow on third line")
	}

	// Should only include content up to limit
	if display2 != "Line 2\n" {
		t.Errorf("Expected display to stop at limit, got '%s'", display2)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/terminal -run TestLineTracker_Overflow -v`

Expected: Test fails because overflow detection isn't working correctly

**Step 3: Fix Track method overflow detection**

The current implementation should work, but let's verify. The issue is that we need to return immediately when overflow occurs.

Review and update `Track` method if needed:

```go
// Track 追踪文本，返回应该显示的文本和是否超限
func (t *LineTracker) Track(text string) (displayText string, overflow bool) {
	if t.stopped {
		return "", false
	}

	var result strings.Builder

	for _, r := range text {
		// 先检查是否已经超限
		if r == '\n' {
			t.lineCount++
			t.currentPos = 0
			result.WriteRune(r)
			// 换行后立即检查
			if t.lineCount > t.maxLines {
				t.stopped = true
				return result.String(), true
			}
			continue
		}

		charWidth := runewidth.RuneWidth(r)

		// 检查是否需要自动换行
		if t.currentPos+charWidth > t.maxWidth {
			t.lineCount++
			t.currentPos = 0
			if t.lineCount > t.maxLines {
				t.stopped = true
				return result.String(), true
			}
		}

		t.currentPos += charWidth
		result.WriteRune(r)
	}

	return result.String(), false
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/terminal -run TestLineTracker_Overflow -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/terminal/tracker.go internal/terminal/tracker_test.go
git commit -m "feat: implement overflow detection in Track"
```

---

## Task 7: Implement LineCount method

**Files:**
- Modify: `internal/terminal/tracker.go`
- Modify: `internal/terminal/tracker_test.go`

**Step 1: Write failing test**

```go
func TestLineTracker_LineCount(t *testing.T) {
	tracker, _ := NewLineTracker(10)

	tracker.Track("Line 1\nLine 2\n")

	if tracker.LineCount() != 3 {
		t.Errorf("Expected LineCount 3, got %d", tracker.LineCount())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/terminal -run TestLineTracker_LineCount -v`

Expected: `undefined: LineTracker.LineCount`

**Step 3: Implement LineCount method**

Add to `LineTracker`:

```go
// LineCount 返回当前追踪的总行数
func (t *LineTracker) LineCount() int {
	return t.lineCount
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/terminal -run TestLineTracker_LineCount -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/terminal/tracker.go internal/terminal/tracker_test.go
git commit -m "feat: add LineCount method"
```

---

## Task 8: Implement Reset method

**Files:**
- Modify: `internal/terminal/tracker.go`
- Modify: `internal/terminal/tracker_test.go`

**Step 1: Write failing test**

```go
func TestLineTracker_Reset(t *testing.T) {
	tracker, _ := NewLineTracker(10)

	tracker.Track("Line 1\nLine 2\n")
	tracker.Reset()

	if tracker.lineCount != 1 {
		t.Errorf("Expected lineCount 1 after reset, got %d", tracker.lineCount)
	}

	if tracker.currentPos != 0 {
		t.Errorf("Expected currentPos 0 after reset, got %d", tracker.currentPos)
	}

	if tracker.stopped {
		t.Error("Expected stopped to be false after reset")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/terminal -run TestLineTracker_Reset -v`

Expected: `undefined: LineTracker.Reset`

**Step 3: Implement Reset method**

Add to `LineTracker`:

```go
// Reset 重置追踪器状态
func (t *LineTracker) Reset() {
	t.currentPos = 0
	t.lineCount = 1
	t.stopped = false
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/terminal -run TestLineTracker_Reset -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/terminal/tracker.go internal/terminal/tracker_test.go
git commit -m "feat: add Reset method"
```

---

## Task 9: Handle zero maxLines (unlimited mode)

**Files:**
- Modify: `internal/terminal/tracker.go`
- Modify: `internal/terminal/tracker_test.go`

**Step 1: Write test for unlimited mode**

```go
func TestLineTracker_UnlimitedMode(t *testing.T) {
	tracker, _ := NewLineTracker(0) // 0 means unlimited

	// Track many lines
	longText := strings.Repeat("Line\n", 100)
	display, overflow := tracker.Track(longText)

	if overflow {
		t.Error("Expected no overflow in unlimited mode")
	}

	if display != longText {
		t.Error("Expected full text to be displayed")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/terminal -run TestLineTracker_UnlimitedMode -v`

Expected: Test fails (overflow occurs)

**Step 3: Modify Track to handle unlimited mode**

Update the `Track` method to check for unlimited mode at the start:

```go
// Track 追踪文本，返回应该显示的文本和是否超限
func (t *LineTracker) Track(text string) (displayText string, overflow bool) {
	if t.stopped {
		return "", false
	}

	// 无限制模式
	if t.maxLines == 0 {
		return text, false
	}

	var result strings.Builder

	// ... rest of the implementation unchanged
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/terminal -run TestLineTracker_UnlimitedMode -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/terminal/tracker.go internal/terminal/tracker_test.go
git commit -m "feat: handle unlimited mode (maxLines=0)"
```

---

## Task 10: Modify REPL to add maxDisplayLines field

**Files:**
- Modify: `internal/terminal/repl.go`
- Modify: `internal/terminal/repl_test.go` (if exists)

**Step 1: Read the current REPL structure**

Run: `grep -A 10 "type REPL struct" internal/terminal/repl.go`

Expected: See current REPL struct definition

**Step 2: Add maxDisplayLines field to REPL**

Modify the `REPL` struct:

```go
// REPL 交互式对话
type REPL struct {
	manager          *conversation.Manager
	conversation     *conversation.Conversation
	renderer         *conversation.Renderer
	stream           bool
	showThinking     bool
	maxDisplayLines  int  // 新增：流式输出最大显示行数
}
```

**Step 3: Update NewREPL constructor**

Modify `NewREPL` function:

```go
// NewREPL 创建 REPL
func NewREPL(manager *conversation.Manager, conv *conversation.Conversation, stream bool, maxDisplayLines int) *REPL {
	return &REPL{
		manager:          manager,
		conversation:     conv,
		stream:           stream,
		showThinking:     true,
		maxDisplayLines:  maxDisplayLines,
	}
}
```

**Step 4: Verify code compiles**

Run: `go build ./internal/terminal`

Expected: No errors (but NewREPL callers will break)

**Step 5: Commit**

```bash
git add internal/terminal/repl.go
git commit -m "feat: add maxDisplayLines field to REPL"
```

---

## Task 11: Modify processStreamChat to use LineTracker

**Files:**
- Modify: `internal/terminal/repl.go`

**Step 1: Find the processStreamChat method**

Run: `grep -n "func (r \*REPL) processStreamChat" internal/terminal/repl.go`

Expected: Line number around 80

**Step 2: Read the current processStreamChat implementation**

Run: `sed -n '80,123p' internal/terminal/repl.go`

Expected: See the streaming chat implementation

**Step 3: Modify processStreamChat to use LineTracker**

Replace the `processStreamChat` method:

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

	// 创建行数追踪器
	tracker, err := NewLineTracker(r.maxDisplayLines)
	if err != nil {
		// 如果创建失败，降级到原始行为
		return r.processStreamChatFallback(input, stream)
	}

	for chunk := range stream {
		fullResponse.WriteString(chunk)

		// 使用追踪器处理显示
		displayText, overflow := tracker.Track(chunk)
		if displayText != "" {
			fmt.Print(displayText)
		}
		if overflow {
			fmt.Print("...")
		}
	}

	// 清除流式输出的原文：上移 lineCount 行并清除
	if tracker.LineCount() > 0 {
		fmt.Printf("\033[%dA\033[J", tracker.LineCount())
	}

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

**Step 4: Add processStreamChatFallback method**

Add this fallback method after `processStreamChat`:

```go
// processStreamChatFallback 降级处理流式对话（当 LineTracker 创建失败时）
func (r *REPL) processStreamChatFallback(input string, stream <-chan string) error {
	var fullResponse strings.Builder
	lineCount := 1

	for chunk := range stream {
		fmt.Print(chunk)
		lineCount += strings.Count(chunk, "\n")
		fullResponse.WriteString(chunk)
	}

	// 清除流式输出的原文
	if lineCount > 0 {
		fmt.Printf("\033[%dA\033[J", lineCount)
	}

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

**Step 5: Verify code compiles**

Run: `go build ./internal/terminal`

Expected: No errors

**Step 6: Commit**

```bash
git add internal/terminal/repl.go
git commit -m "feat: integrate LineTracker into processStreamChat"
```

---

## Task 12: Update chat command to pass maxDisplayLines

**Files:**
- Modify: `cmd/tada/chat.go`

**Step 1: Find chat command initialization**

Run: `grep -n "NewREPL" cmd/tada/chat.go`

Expected: Line number where REPL is created

**Step 2: Read the chat command context**

Run: `cat cmd/tada/chat.go`

Expected: See the full chat command implementation

**Step 3: Find config loading and maxDisplayLines usage**

Run: `grep -n "maxDisplayLines\|MaxDisplayLines\|cfg.Chat" cmd/tada/chat.go`

Expected: May not exist yet

**Step 4: Add maxDisplayLines variable before NewREPL call**

Find the line where `NewREPL` is called and add before it:

```go
// 从配置读取流式输出行数限制
maxDisplayLines := 10 // 默认值
if cfg.Chat.Streaming.MaxDisplayLines > 0 {
	maxDisplayLines = cfg.Chat.Streaming.MaxDisplayLines
}
```

**Step 5: Update NewREPL call**

Find and modify the `NewREPL` call:

```go
// 旧代码:
repl := terminal.NewREPL(manager, conv, *stream)

// 新代码:
repl := terminal.NewREPL(manager, conv, *stream, maxDisplayLines)
```

**Step 6: Verify code compiles**

Run: `go build ./cmd/tada`

Expected: No errors

**Step 7: Commit**

```bash
git add cmd/tada/chat.go
git commit -m "feat: wire up maxDisplayLines config in chat command"
```

---

## Task 13: Add integration test for limited streaming

**Files:**
- Create: `tests/integration/limited_streaming_test.go`

**Step 1: Create integration test file**

```go
// tests/integration/limited_streaming_test.go
// +build integration

package integration

import (
	"strings"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/terminal"
)

func TestLineTracker_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("LimitedMode", func(t *testing.T) {
		tracker, err := terminal.NewLineTracker(5)
		if err != nil {
			t.Fatalf("Failed to create tracker: %v", err)
		}

		// 模拟流式输入
		chunks := []string{
			"Line 1\n",
			"Line 2\n",
			"Line 3\n",
			"Line 4\n",
			"Line 5\n",
			"Line 6\n", // 应该触发 overflow
		}

		for i, chunk := range chunks {
			display, overflow := tracker.Track(chunk)
			t.Logf("Chunk %d: display='%s', overflow=%v", i, display, overflow)

			if i < 5 && overflow {
				t.Errorf("Chunk %d: unexpected overflow", i)
			}
			if i >= 5 && !overflow {
				t.Errorf("Chunk %d: expected overflow", i)
			}
		}

		t.Logf("Final lineCount: %d", tracker.LineCount())
	})

	t.Run("UnlimitedMode", func(t *testing.T) {
		tracker, err := terminal.NewLineTracker(0)
		if err != nil {
			t.Fatalf("Failed to create tracker: %v", err)
		}

		longText := strings.Repeat("This is a line\n", 100)
		display, overflow := tracker.Track(longText)

		if overflow {
			t.Error("Expected no overflow in unlimited mode")
		}

		if display != longText {
			t.Error("Expected full text in unlimited mode")
		}
	})
}
```

**Step 2: Run integration test**

Run: `TADA_INTEGRATION_TEST=1 go test ./tests/integration -run TestLineTracker_Integration -v`

Expected: `PASS`

**Step 3: Commit**

```bash
git add tests/integration/limited_streaming_test.go
git commit -m "test: add integration test for limited streaming"
```

---

## Task 14: Update example config

**Files:**
- Check: `~/.tada/config.yaml` (user's local file, not in repo)
- Create or Update: `docs/getting-started.md` (documentation)

**Step 1: Check if example config exists in repo**

Run: `find . -name "config.yaml" -o -name "*config*example*" | grep -v node_modules`

Expected: May find example config or need to document in getting-started.md

**Step 2: Update getting-started.md with new config**

Run: `cat docs/getting-started.md`

Expected: See current documentation

**Step 3: Add streaming config documentation**

Find the config section in getting-started.md and add:

```markdown
### Chat Configuration

```yaml
chat:
  streaming:
    max_display_lines: 10  # 流式输出最大显示行数（默认 10）
                            # 设为 0 表示不限制行数
```

- `max_display_lines`: 控制流式输出时显示的最大行数
  - 超过此行数后，流式输出将停止并显示 "..."
  - 流式结束后会清除原文并渲染完整的 markdown
  - 设为 0 表示不限制，完全流式输出
```

**Step 4: Commit**

```bash
git add docs/getting-started.md
git commit -m "docs: add streaming config documentation"
```

---

## Task 15: Manual testing and verification

**Files:** N/A (manual testing)

**Step 1: Build the project**

Run: `go build -o tada cmd/tada/main.go`

Expected: Binary created successfully

**Step 2: Test with limited lines (default 10)**

Run: `./tada chat`

Enter: `"写一段关于 Go 语言历史的介绍，要求至少 20 行"`

Verify:
- Streaming output shows first ~10 lines then "..."
- After streaming completes, original text is cleared
- Rendered markdown appears cleanly

**Step 3: Test with unlimited mode**

Edit `~/.tada/config.yaml`:

```yaml
chat:
  streaming:
    max_display_lines: 0
```

Run: `./tada chat`

Enter: `"写一首诗"`

Verify:
- Full streaming output without "..."
- All original text cleared and rendered

**Step 4: Test with small limit**

Edit `~/.tada/config.yaml`:

```yaml
chat:
  streaming:
    max_display_lines: 3
```

Run: `./tada chat`

Enter: `"列出 5 个 Go 的特性"`

Verify:
- Only first 3 lines shown during streaming
- "..." appears early
- Clean render at end

**Step 5: Test short response**

Run: `./tada chat`

Enter: `"你好"`

Verify:
- No overflow for short response
- Clean render at end

**Step 6: Test code blocks**

Run: `./tada chat`

Enter: `"写一个 Go hello world 程序"`

Verify:
- Code block renders correctly
- No leftover raw text

**Step 7: Document results**

Create a note of any issues found and fix them.

**Step 8: Commit documentation**

```bash
git add docs/plans/2026-02-24-limited-streaming-rendering-implementation.md
git commit -m "docs: complete limited streaming implementation plan"
```

---

## Testing Strategy Summary

1. **Unit Tests**: Each `LineTracker` method has dedicated tests
2. **Integration Tests**: Full workflow with different modes
3. **Manual Tests**: Real-world usage scenarios with different configs

## Dependencies Summary

| Dependency | Purpose | Added In |
|------------|---------|----------|
| `github.com/mattn/go-runewidth` | Character width calculation | Task 1 |
| `golang.org/x/term` | Terminal width (already in use) | - |

## Notes for Implementation

- The `LineTracker` counts lines starting from 1 (first line is already active)
- Automatic line wrap is detected when `currentPos + charWidth > maxWidth`
- The overflow check happens immediately after line count increases
- In unlimited mode (maxLines=0), we bypass all tracking logic
- The fallback in `processStreamChatFallback` preserves original behavior if tracker creation fails
