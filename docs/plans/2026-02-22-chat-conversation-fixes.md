# Chat Conversation Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement missing features (--no-history, --today, /prompt) and fix documentation/logging issues

**Architecture:**
- `--no-history`: Skip storage.Save() calls when flag is set, use in-memory conversation
- `--today`: Filter conversations by CreatedAt date in runListConversations
- `/prompt`: Change conversation's PromptName and reload system message via Manager
- Add godoc comments to exported functions
- Add log.Printf for prompt load failures

**Tech Stack:** Go 1.23+, cobra flags, standard library log

---

## Task 1: Add ChatStream Godoc Comment

**Files:**
- Modify: `internal/conversation/manager.go:109`

**Step 1: Read current ChatStream function**

Run: `head -n 165 internal/conversation/manager.go | tail -n 60`

Expected: See ChatStream function at line 109 without godoc comment

**Step 2: Add godoc comment before ChatStream**

Edit `internal/conversation/manager.go`, add before line 109:

```go
// ChatStream 发送消息并流式获取回复
//
// 流式处理流程：
// 1. 添加用户消息到对话
// 2. 调用 AI 提供者的流式接口
// 3. 在独立 goroutine 中：
//    - 逐块发送响应到输出 channel
//    - 完成后重新加载对话（避免竞态条件）
//    - 添加助手消息并保存
//
// 参数：
//   convID - 对话 ID
//   userInput - 用户输入内容
//
// 返回：
//   <-chan string - 响应内容流，消费完后 channel 自动关闭
//   error - 错误信息（nil 表示成功）
func (m *Manager) ChatStream(convID string, userInput string) (<-chan string, error) {
```

**Step 3: Verify syntax**

Run: `go build ./internal/conversation/...`

Expected: No errors

**Step 4: Commit**

```bash
git add internal/conversation/manager.go
git commit -m "docs: add godoc comment for ChatStream"
```

---

## Task 2: Add Logging for Prompt Load Failures

**Files:**
- Modify: `internal/conversation/manager.go:34-46`

**Step 1: Add log import**

Add to imports in `internal/conversation/manager.go`:

```go
import (
    "context"
    "fmt"
    "log"  // Add this
    "strings"
    "time"

    "github.com/Lin-Jiong-HDU/tada/internal/ai"
)
```

**Step 2: Add log message in Create method**

Modify the prompt load error handling section (around line 34-46):

```go
// 加载 prompt 模板
prompt, err := m.promptLoader.Load(promptName)
if err != nil {
    log.Printf("Warning: failed to load prompt '%s': %v, using default", promptName, err)
    // 如果加载失败，使用默认 prompt
    conv.AddMessage(Message{
        Role:    "system",
        Content: "You are a helpful assistant.",
    })
} else {
    conv.AddMessage(Message{
        Role:    "system",
        Content: prompt.SystemPrompt,
    })
}
```

**Step 3: Test the logging**

Run: `go test ./internal/conversation/... -v`

Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/conversation/manager.go
git commit -m "feat: log prompt load failures with fallback message"
```

---

## Task 3: Add Ephemeral Conversation Type for --no-history

**Files:**
- Modify: `internal/conversation/types.go`
- Test: `internal/conversation/types_test.go` (create new)

**Step 1: Write failing test for ephemeral conversation**

Create `internal/conversation/types_test.go`:

```go
package conversation

import (
    "testing"
    "time"
)

func TestConversation_IsEphemeral(t *testing.T) {
    conv := NewConversation("default")
    conv.Name = "test"

    // Default is not ephemeral
    if conv.IsEphemeral() {
        t.Error("Expected default conversation to not be ephemeral")
    }

    // Set as ephemeral
    conv.SetEphemeral(true)
    if !conv.IsEphemeral() {
        t.Error("Expected conversation to be ephemeral after SetEphemeral(true)")
    }

    // Messages can still be added
    conv.AddMessage(Message{
        Role:      "user",
        Content:   "test",
        Timestamp: time.Now(),
    })

    if len(conv.Messages) != 1 {
        t.Errorf("Expected 1 message, got %d", len(conv.Messages))
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/conversation -run TestConversation_IsEphemeral -v`

Expected: FAIL with "undefined: IsEphemeral"

**Step 3: Implement ephemeral support in types.go**

Add to `internal/conversation/types.go` after the Conversation struct:

```go
// Conversation 表示一个对话
type Conversation struct {
    ID         string             `json:"id"`
    Name       string             `json:"name"`
    PromptName string             `json:"prompt_name"`
    Messages   []Message          `json:"messages"`
    Status     ConversationStatus `json:"status"`
    CreatedAt  time.Time          `json:"created_at"`
    UpdatedAt  time.Time          `json:"updated_at"`
    ephemeral  bool               `json:"-"` // 不保存到文件，不记录历史
}
```

Add methods after AddMessage:

```go
// IsEphemeral 返回对话是否为临时模式（不保存历史）
func (c *Conversation) IsEphemeral() bool {
    return c.ephemeral
}

// SetEphemeral 设置对话为临时模式
func (c *Conversation) SetEphemeral(ephemeral bool) {
    c.ephemeral = ephemeral
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/conversation -run TestConversation_IsEphemeral -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/conversation/types.go internal/conversation/types_test.go
git commit -m "feat: add ephemeral conversation type for --no-history"
```

---

## Task 4: Update Manager to Support Ephemeral Conversations

**Files:**
- Modify: `internal/conversation/manager.go`
- Test: `internal/conversation/manager_test.go` (create new)

**Step 1: Write failing test for ephemeral conversation**

Create `internal/conversation/manager_test.go`:

```go
package conversation

import (
    "testing"
    "time"

    "github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// mockAIProviderEphemeral 用于测试的 mock AI provider
type mockAIProviderEphemeral struct{}

func (m *mockAIProviderEphemeral) Chat(ctx context.Context, messages []ai.Message) (string, error) {
    return "Mock response", nil
}

func (m *mockAIProviderEphemeral) ChatStream(ctx context.Context, messages []ai.Message) (<-chan string, error) {
    out := make(chan string)
    go func() {
        defer close(out)
        out <- "Mock stream"
    }()
    return out, nil
}

func TestManager_CreateEphemeral(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "conv-test-*")
    if err != nil {
        t.Fatal(err)
    }
    defer os.RemoveAll(tmpDir)

    storage := NewFileStorage(tmpDir)
    promptDir, _ := os.MkdirTemp("", "prompt-test-*")
    loader := NewPromptLoader(promptDir)

    manager := NewManager(storage, loader, &mockAIProviderEphemeral{})

    // Create ephemeral conversation
    conv, err := manager.CreateEphemeral("test", "default")
    if err != nil {
        t.Fatalf("CreateEphemeral failed: %v", err)
    }

    if !conv.IsEphemeral() {
        t.Error("Expected conversation to be ephemeral")
    }

    // Should not be saved to storage
    _, err = storage.Get(conv.ID)
    if err == nil {
        t.Error("Expected ephemeral conversation to not be saved")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/conversation -run TestManager_CreateEphemeral -v`

Expected: FAIL with "undefined: CreateEphemeral"

**Step 3: Implement CreateEphemeral in manager.go**

Add to `internal/conversation/manager.go` after Create method:

```go
// CreateEphemeral 创建临时对话（不保存历史）
func (m *Manager) CreateEphemeral(name, promptName string) (*Conversation, error) {
    conv := NewConversation(promptName)
    conv.Name = name
    conv.SetEphemeral(true) // 标记为临时对话

    // 加载 prompt 模板
    prompt, err := m.promptLoader.Load(promptName)
    if err != nil {
        log.Printf("Warning: failed to load prompt '%s': %v, using default", promptName, err)
        conv.AddMessage(Message{
            Role:    "system",
            Content: "You are a helpful assistant.",
        })
    } else {
        conv.AddMessage(Message{
            Role:    "system",
            Content: prompt.SystemPrompt,
        })
    }

    // 不保存到存储
    return conv, nil
}
```

**Step 4: Update Chat methods to skip save for ephemeral**

Modify Chat method in manager.go, replace the save section:

```go
// 保存（临时对话不保存）
if !conv.IsEphemeral() {
    if err := m.storage.Save(conv); err != nil {
        return "", fmt.Errorf("failed to save conversation: %w", err)
    }
}
```

Modify ChatStream method in manager.go, replace the save section:

```go
// 保存（临时对话不保存）
if !reloadedConv.IsEphemeral() {
    _ = m.storage.Save(reloadedConv)
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/conversation -v`

Expected: All tests pass

**Step 6: Commit**

```bash
git add internal/conversation/manager.go internal/conversation/manager_test.go
git commit -m "feat: add CreateEphemeral and skip save for ephemeral conversations"
```

---

## Task 5: Wire --no-history Flag in chat.go

**Files:**
- Modify: `cmd/tada/chat.go:105-121`

**Step 1: Update runChat to use CreateEphemeral**

Modify the conversation creation section in `cmd/tada/chat.go`:

```go
// 创建或恢复对话
var conv *conversation.Conversation
var err error

if chatContinueID != "" {
    conv, err = manager.Get(chatContinueID)
    if err != nil {
        return fmt.Errorf("对话不存在: %s", chatContinueID)
    }
    fmt.Printf("📂 恢复对话: %s (%s)\n", conv.ID, conv.PromptName)
} else if chatNoHistory {
    // 使用临时对话，不保存历史
    conv, err = manager.CreateEphemeral(chatName, chatPromptName)
    if err != nil {
        return fmt.Errorf("创建临时对话失败: %w", err)
    }
    fmt.Printf("📝 临时对话 (%s) - 不保存历史\n", conv.PromptName)
} else {
    conv, err = manager.Create(chatName, chatPromptName)
    if err != nil {
        return fmt.Errorf("创建对话失败: %w", err)
    }
    fmt.Printf("📝 新对话 (%s)\n", conv.PromptName)
}
```

**Step 2: Test the flag**

Run: `go build -o tada cmd/tada/main.go && ./tada chat --no-history`

Expected: "临时对话" message appears

**Step 3: Commit**

```bash
git add cmd/tada/chat.go
git commit -m "feat: implement --no-history flag with ephemeral conversations"
```

---

## Task 6: Implement --today Filter for Conversation List

**Files:**
- Modify: `internal/conversation/storage.go`
- Modify: `cmd/tada/chat.go:173-197`

**Step 1: Add ListToday method to Storage interface and FileStorage**

Modify `internal/conversation/storage.go`, update Storage interface:

```go
// Storage 对话存储接口
type Storage interface {
    Save(conv *Conversation) error
    Get(id string) (*Conversation, error)
    List() ([]*Conversation, error)
    ListToday() ([]*Conversation, error) // 新增
    Delete(id string) error
}
```

Add ListToday method to FileStorage:

```go
// ListToday 列出今天的对话
func (s *FileStorage) ListToday() ([]*Conversation, error) {
    var conversations []*Conversation

    // 获取今天的日期路径
    todayPath := filepath.Join(s.conversationsDir, time.Now().Format("20060102"))

    entries, err := os.ReadDir(todayPath)
    if err != nil {
        if os.IsNotExist(err) {
            return conversations, nil // 今天还没有对话
        }
        return nil, fmt.Errorf("failed to read today's conversations directory: %w", err)
    }

    for _, convEntry := range entries {
        if !convEntry.IsDir() {
            continue
        }

        conv, err := s.Get(convEntry.Name())
        if err != nil {
            continue
        }

        conversations = append(conversations, conv)
    }

    // 按更新时间排序（最新的在前）
    sort.Slice(conversations, func(i, j int) bool {
        return conversations[i].UpdatedAt.After(conversations[j].UpdatedAt)
    })

    return conversations, nil
}
```

Add import for time package if not present:
```go
import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "time" // Add this
)
```

**Step 2: Add ListToday to Manager**

Add to `internal/conversation/manager.go`:

```go
// ListToday 列出今天的对话
func (m *Manager) ListToday() ([]*Conversation, error) {
    return m.storage.ListToday()
}
```

**Step 3: Update runListConversations to use ListToday when flag is set**

Modify `cmd/tada/chat.go`, update runListConversations function:

```go
func runListConversations(manager *conversation.Manager) error {
    var convs []*conversation.Conversation
    var err error

    if chatToday {
        convs, err = manager.ListToday()
    } else {
        convs, err = manager.List()
    }

    if err != nil {
        return err
    }

    if len(convs) == 0 {
        if chatToday {
            fmt.Println("💬 今天没有对话记录")
        } else {
            fmt.Println("💬 没有对话记录")
        }
        return nil
    }

    fmt.Println("💬 对话历史:")
    fmt.Println()

    for _, conv := range convs {
        fmt.Printf("  %s  [%s]  %d 条消息  %s\n",
            conv.ID[:12],
            conv.PromptName,
            len(conv.Messages),
            conv.UpdatedAt.Format("2006-01-02 15:04"),
        )
    }

    return nil
}
```

**Step 4: Test the --today flag**

Run:
```bash
# Create a conversation
./tada chat -n "test" <<< "/exit"

# List all
./tada chat -l

# List today's only
./tada chat --today -l
```

Expected: --today shows only today's conversations

**Step 5: Write tests**

Create `internal/conversation/storage_test.go`:

```go
package conversation

import (
    "os"
    "path/filepath"
    "testing"
    "time"
)

func TestFileStorage_ListToday(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "storage-test-*")
    if err != nil {
        t.Fatal(err)
    }
    defer os.RemoveAll(tmpDir)

    storage := NewFileStorage(tmpDir)

    // Create a conversation for today
    conv := NewConversation("default")
    conv.Name = "today-test"
    if err := storage.Save(conv); err != nil {
        t.Fatal(err)
    }

    // List today's conversations
    convs, err := storage.ListToday()
    if err != nil {
        t.Fatalf("ListToday failed: %v", err)
    }

    if len(convs) != 1 {
        t.Errorf("Expected 1 conversation, got %d", len(convs))
    }

    if convs[0].ID != conv.ID {
        t.Errorf("Expected conversation ID %s, got %s", conv.ID, convs[0].ID)
    }
}
```

**Step 6: Run tests**

Run: `go test ./internal/conversation -v`

Expected: All tests pass

**Step 7: Commit**

```bash
git add internal/conversation/storage.go internal/conversation/storage_test.go
git add internal/conversation/manager.go
git add cmd/tada/chat.go
git commit -m "feat: implement --today flag for filtering conversations"
```

---

## Task 7: Implement /prompt Command

**Files:**
- Modify: `internal/conversation/types.go`
- Modify: `internal/conversation/manager.go`
- Modify: `internal/terminal/repl.go:135-141`

**Step 1: Add SwitchPrompt method to Conversation**

Add to `internal/conversation/types.go` after SetEphemeral:

```go
// SwitchPrompt 切换对话的 prompt 模板
// 替换系统消息为新的 prompt，保留用户和助手的对话历史
func (c *Conversation) SwitchPrompt(newPromptName string, newSystemPrompt string) {
    // 移除旧的系统消息（第一条消息）
    if len(c.Messages) > 0 && c.Messages[0].Role == "system" {
        c.Messages = c.Messages[1:]
    }

    // 在开头插入新的系统消息
    c.PromptName = newPromptName
    systemMsg := Message{
        Role:      "system",
        Content:   newSystemPrompt,
        Timestamp: time.Now(),
    }
    c.Messages = append([]Message{systemMsg}, c.Messages...)
    c.UpdatedAt = time.Now()
}
```

**Step 2: Add SwitchPrompt method to Manager**

Add to `internal/conversation/manager.go`:

```go
// SwitchPrompt 切换对话的 prompt 模板
func (m *Manager) SwitchPrompt(convID, newPromptName string) error {
    conv, err := m.Get(convID)
    if err != nil {
        return fmt.Errorf("conversation not found: %w", err)
    }

    // 加载新的 prompt
    prompt, err := m.promptLoader.Load(newPromptName)
    if err != nil {
        log.Printf("Warning: failed to load prompt '%s': %v", newPromptName, err)
        return fmt.Errorf("failed to load prompt '%s': %w", newPromptName, err)
    }

    // 切换 prompt
    conv.SwitchPrompt(newPromptName, prompt.SystemPrompt)

    // 保存（临时对话不保存）
    if !conv.IsEphemeral() {
        if err := m.storage.Save(conv); err != nil {
            return fmt.Errorf("failed to save conversation: %w", err)
        }
    }

    return nil
}

// ListPrompts 列出所有可用的 prompt 模板
func (m *Manager) ListPrompts() ([]*PromptTemplate, error) {
    return m.promptLoader.List()
}
```

**Step 3: Update REPL HandleCommand for /prompt**

Modify `internal/terminal/repl.go`, update the /prompt case:

```go
case "/prompt":
    if len(parts) < 2 {
        // 没有参数，列出可用的 prompts
        r.DisplayAvailablePrompts()
        return false, nil
    }

    // 切换 prompt
    if err := r.manager.SwitchPrompt(r.conversation.ID, parts[1]); err != nil {
        fmt.Printf("切换 prompt 失败: %v\n", err)
        return false, nil
    }

    fmt.Printf("✓ 已切换到 prompt: %s\n", parts[1])
    return false, nil
```

Add DisplayAvailablePrompts method:

```go
// DisplayAvailablePrompts 显示可用的 prompt 模板
func (r *REPL) DisplayAvailablePrompts() {
    prompts, err := r.manager.ListPrompts()
    if err != nil {
        fmt.Printf("获取 prompt 列表失败: %v\n", err)
        return
    }

    if len(prompts) == 0 {
        fmt.Println("没有可用的 prompt 模板")
        return
    }

    fmt.Println("\n可用的 prompt 模板:")
    for _, p := range prompts {
        if p.Title != "" {
            fmt.Printf("  • %s - %s\n", p.Name, p.Title)
        } else {
            fmt.Printf("  • %s\n", p.Name)
        }
        if p.Description != "" {
            fmt.Printf("    %s\n", p.Description)
        }
    }
    fmt.Println()
}
```

Update DisplayHelp:

```go
// DisplayHelp 显示帮助
func (r *REPL) DisplayHelp() {
    help := `
可用命令:
  /help              显示此帮助
  /clear             清屏
  /prompt [name]     切换/列出 prompt 模板
  /exit, /quit       退出并保存
`
    fmt.Println(help)
}
```

**Step 4: Test /prompt command**

Run:
```bash
./tada chat
# In chat session:
/prompt
/prompt coder
```

Expected: Lists prompts and switches successfully

**Step 5: Write tests**

Add to `internal/conversation/types_test.go`:

```go
func TestConversation_SwitchPrompt(t *testing.T) {
    conv := NewConversation("default")
    conv.AddMessage(Message{Role: "system", Content: "Old prompt", Timestamp: time.Now()})
    conv.AddMessage(Message{Role: "user", Content: "Hello", Timestamp: time.Now()})

    conv.SwitchPrompt("coder", "You are a coding expert.")

    if conv.PromptName != "coder" {
        t.Errorf("Expected PromptName 'coder', got '%s'", conv.PromptName)
    }

    if len(conv.Messages) != 2 {
        t.Errorf("Expected 2 messages, got %d", len(conv.Messages))
    }

    if conv.Messages[0].Content != "You are a coding expert." {
        t.Errorf("Expected new system prompt, got '%s'", conv.Messages[0].Content)
    }

    if conv.Messages[1].Content != "Hello" {
        t.Errorf("Expected user message preserved, got '%s'", conv.Messages[1].Content)
    }
}
```

**Step 6: Run tests**

Run: `go test ./internal/conversation -v`

Expected: All tests pass

**Step 7: Commit**

```bash
git add internal/conversation/types.go internal/conversation/types_test.go
git add internal/conversation/manager.go internal/conversation/manager_test.go
git add internal/terminal/repl.go
git commit -m "feat: implement /prompt command for switching prompt templates"
```

---

## Task 8: Update Documentation

**Files:**
- Modify: `cmd/tada/chat.go:34-37` (update command description)
- Modify: `README.md` (if exists, add new features)

**Step 1: Update chat command help text**

Update the Long description in `cmd/tada/chat.go`:

```go
cmd := &cobra.Command{
    Use:   "chat",
    Short: "与 AI 对话",
    Long: `交互式 AI 对话，支持多轮对话、历史记录和自定义 prompt

特性:
  - 多轮对话: 自动保存对话历史
  - Prompt 模板: 支持 /prompt 命令切换
  - 临时模式: 使用 --no-history 不保存历史
  - 流式输出: 实时显示 AI 响应
  - Markdown 渲染: 美化输出格式
`,
    // ...
}
```

**Step 2: Build and verify help**

Run: `./tada chat --help`

Expected: Updated help text appears

**Step 3: Commit**

```bash
git add cmd/tada/chat.go
git commit -m "docs: update chat command help text with new features"
```

---

## Task 9: Final Verification

**Step 1: Run all tests**

Run: `go test ./... -v`

Expected: All tests pass

**Step 2: Build**

Run: `go build -o tada cmd/tada/main.go`

Expected: No errors

**Step 3: Manual test --no-history**

Run:
```bash
./tada chat --no-history -n "ephemeral-test"
# Enter: hello
# Enter: /exit
./tada chat -l
```

Expected: "ephemeral-test" does NOT appear in list

**Step 4: Manual test --today**

Run:
```bash
./tada chat --today -l
```

Expected: Only today's conversations shown

**Step 5: Manual test /prompt**

Run:
```bash
./tada chat
# Enter: /prompt
# Enter: /prompt coder
```

Expected: Lists prompts and switches successfully

**Step 6: Final commit**

```bash
git add -A
git commit -m "chore: final verification of chat conversation fixes"
```

---

## Summary

This plan implements:
1. ✅ Godoc comment for ChatStream (Task 1)
2. ✅ Logging for prompt load failures (Task 2)
3. ✅ --no-history flag (Tasks 3-5)
4. ✅ --today flag (Task 6)
5. ✅ /prompt command (Task 7)
6. ✅ Documentation updates (Tasks 8-9)

Total: 9 tasks, each following TDD with test → implement → verify → commit cycle.
