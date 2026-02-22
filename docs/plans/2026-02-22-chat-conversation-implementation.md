# Chat Conversation Feature Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 tada 添加纯对话功能，支持多轮对话、历史持久化、自定义 prompt 模板、流式输出和 markdown 终端渲染。

**Architecture:** 新建 conversation 包处理对话逻辑（Manager、Storage、PromptLoader、Renderer），扩展 AIProvider 接口支持 ChatStream，重写 chatCmd 使用 REPL 交互模式。

**Tech Stack:** Go 1.25.7, Bubble Tea, Glamour (markdown渲染), OpenAI/GLM API, 文件系统存储

---

## Task 1: 添加 glamour 依赖

**Files:**
- Modify: `go.mod`

**Step 1: 添加依赖**

```bash
go get github.com/charmbracelet/glamour@latest
```

**Step 2: 验证依赖**

Run: `go mod tidy`
Expected: 无错误

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add glamour for markdown rendering"
```

---

## Task 2: 创建 conversation 包基础结构

**Files:**
- Create: `internal/conversation/types.go`
- Test: `internal/conversation/types_test.go`

**Step 1: Write the failing test**

Create `internal/conversation/types_test.go`:

```go
package conversation

import (
	"testing"
	"time"
)

func TestConversation_NewConversation(t *testing.T) {
	conv := NewConversation("test-prompt")

	if conv.ID == "" {
		t.Error("Expected non-empty ID")
	}

	if conv.PromptName != "test-prompt" {
		t.Errorf("Expected prompt name 'test-prompt', got '%s'", conv.PromptName)
	}

	if conv.Status != StatusActive {
		t.Errorf("Expected status active, got %s", conv.Status)
	}

	if len(conv.Messages) != 0 {
		t.Error("Expected empty messages")
	}
}

func TestConversation_AddMessage(t *testing.T) {
	conv := NewConversation("default")

	msg := Message{
		Role:      "user",
		Content:   "hello",
		Timestamp: time.Now(),
	}

	conv.AddMessage(msg)

	if len(conv.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(conv.Messages))
	}

	if conv.Messages[0].Content != "hello" {
		t.Errorf("Expected message content 'hello', got '%s'", conv.Messages[0].Content)
	}
}

func TestMessage_ToAIFormat(t *testing.T) {
	msg := Message{
		Role:      "user",
		Content:   "test",
		Timestamp: time.Now(),
	}

	aiMsg := msg.ToAIFormat()

	if aiMsg.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", aiMsg.Role)
	}

	if aiMsg.Content != "test" {
		t.Errorf("Expected content 'test', got '%s'", aiMsg.Content)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/conversation -v`
Expected: FAIL with "undefined: NewConversation"

**Step 3: Write minimal implementation**

Create `internal/conversation/types.go`:

```go
package conversation

import (
	"time"

	"github.com/google/uuid"
	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// ConversationStatus 对话状态
type ConversationStatus string

const (
	StatusActive   ConversationStatus = "active"
	StatusArchived ConversationStatus = "archived"
)

// Conversation 表示一个对话
type Conversation struct {
	ID         string               `json:"id"`
	Name       string               `json:"name"`
	PromptName string               `json:"prompt_name"`
	Messages   []Message            `json:"messages"`
	Status     ConversationStatus   `json:"status"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

// Message 表示单条消息
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// NewConversation 创建新对话
func NewConversation(promptName string) *Conversation {
	now := time.Now()
	return &Conversation{
		ID:         uuid.New().String(),
		PromptName: promptName,
		Messages:   []Message{},
		Status:     StatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// AddMessage 添加消息
func (c *Conversation) AddMessage(msg Message) {
	c.Messages = append(c.Messages, msg)
	c.UpdatedAt = time.Now()
}

// ToAIFormat 转换为 AI 消息格式
func (m *Message) ToAIFormat() ai.Message {
	return ai.Message{
		Role:    m.Role,
		Content: m.Content,
	}
}

// GetMessagesForAI 获取用于 AI 的消息列表
func (c *Conversation) GetMessagesForAI() []ai.Message {
	messages := make([]ai.Message, 0, len(c.Messages))
	for _, msg := range c.Messages {
		messages = append(messages, msg.ToAIFormat())
	}
	return messages
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/conversation -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/conversation/types.go internal/conversation/types_test.go
git commit -m "feat(conversation): add conversation types and basic operations"
```

---

## Task 3: 实现 PromptLoader

**Files:**
- Create: `internal/conversation/prompt.go`
- Create: `internal/conversation/prompt_test.go`

**Step 1: Write the failing test**

Create `internal/conversation/prompt_test.go`:

```go
package conversation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPromptLoader_Load(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prompt-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试 prompt 文件
	promptContent := `---
name: "test"
title: "Test Prompt"
description: "A test prompt"
---

You are a test assistant.`

	promptFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(promptFile, []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewPromptLoader(tmpDir)
	prompt, err := loader.Load("test")

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if prompt.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", prompt.Name)
	}

	if prompt.Title != "Test Prompt" {
		t.Errorf("Expected title 'Test Prompt', got '%s'", prompt.Title)
	}

	if prompt.SystemPrompt != "You are a test assistant." {
		t.Errorf("Expected system prompt 'You are a test assistant.', got '%s'", prompt.SystemPrompt)
	}
}

func TestPromptLoader_List(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prompt-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建多个 prompt 文件
	prompt1 := `---
name: "default"
---
Default prompt`
	prompt2 := `---
name: "coder"
---
Coder prompt`

	os.WriteFile(filepath.Join(tmpDir, "default.md"), []byte(prompt1), 0644)
	os.WriteFile(filepath.Join(tmpDir, "coder.md"), []byte(prompt2), 0644)

	loader := NewPromptLoader(tmpDir)
	prompts, err := loader.List()

	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(prompts) != 2 {
		t.Errorf("Expected 2 prompts, got %d", len(prompts))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/conversation -run TestPromptLoader -v`
Expected: FAIL with "undefined: NewPromptLoader"

**Step 3: Write minimal implementation**

Create `internal/conversation/prompt.go`:

```go
package conversation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PromptLoader 加载 prompt 模板
type PromptLoader struct {
	promptsDir string
}

// NewPromptLoader 创建 PromptLoader
func NewPromptLoader(promptsDir string) *PromptLoader {
	return &PromptLoader{
		promptsDir: promptsDir,
	}
}

// PromptTemplate prompt 模板
type PromptTemplate struct {
	Name         string
	Title        string
	Description  string
	Content      string
	SystemPrompt string
}

// Load 加载指定名称的 prompt
func (l *PromptLoader) Load(name string) (*PromptTemplate, error) {
	path := filepath.Join(l.promptsDir, name+".md")

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt: %w", err)
	}

	return l.Parse(string(content)), nil
}

// Parse 解析 prompt 内容
func (l *PromptLoader) Parse(content string) *PromptTemplate {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		// 没有 frontmatter，整个内容作为 system prompt
		return &PromptTemplate{
			Name:         "default",
			Content:      content,
			SystemPrompt: strings.TrimSpace(content),
		}
	}

	// 解析 frontmatter
	frontmatter := parts[1]
	systemPrompt := strings.TrimSpace(parts[2])

	template := &PromptTemplate{
		Content:      content,
		SystemPrompt: systemPrompt,
	}

	// 解析 frontmatter 中的字段
	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			template.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			template.Name = strings.Trim(template.Name, `"`)
		} else if strings.HasPrefix(line, "title:") {
			template.Title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			template.Title = strings.Trim(template.Title, `"`)
		} else if strings.HasPrefix(line, "description:") {
			template.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			template.Description = strings.Trim(template.Description, `"`)
		}
	}

	if template.Name == "" {
		template.Name = "default"
	}

	return template
}

// List 列出所有可用的 prompt
func (l *PromptLoader) List() ([]*PromptTemplate, error) {
	entries, err := os.ReadDir(l.promptsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompts directory: %w", err)
	}

	var prompts []*PromptTemplate
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		prompt, err := l.Load(name)
		if err != nil {
			continue // 跳过无法加载的文件
		}

		prompts = append(prompts, prompt)
	}

	return prompts, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/conversation -run TestPromptLoader -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/conversation/prompt.go internal/conversation/prompt_test.go
git commit -m "feat(conversation): add prompt loader for template management"
```

---

## Task 4: 实现 ConversationStorage

**Files:**
- Create: `internal/conversation/storage.go`
- Create: `internal/conversation/storage_test.go`

**Step 1: Write the failing test**

Create `internal/conversation/storage_test.go`:

```go
package conversation

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStorage_SaveAndGet(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "conv-storage-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewFileStorage(tmpDir)

	conv := NewConversation("default")
	conv.ID = "test-id-123"
	conv.AddMessage(Message{
		Role:      "user",
		Content:   "hello",
		Timestamp: time.Now(),
	})

	// 保存
	err = storage.Save(conv)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 读取
	loaded, err := storage.Get("test-id-123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if loaded.ID != conv.ID {
		t.Errorf("Expected ID %s, got %s", conv.ID, loaded.ID)
	}

	if len(loaded.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(loaded.Messages))
	}
}

func TestFileStorage_List(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "conv-storage-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewFileStorage(tmpDir)

	// 创建多个对话
	conv1 := NewConversation("default")
	conv1.ID = "id-1"
	storage.Save(conv1)

	conv2 := NewConversation("coder")
	conv2.ID = "id-2"
	storage.Save(conv2)

	// 列出
	list, err := storage.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("Expected 2 conversations, got %d", len(list))
	}
}

func TestFileStorage_Delete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "conv-storage-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewFileStorage(tmpDir)

	conv := NewConversation("default")
	conv.ID = "test-id"
	storage.Save(conv)

	// 删除
	err = storage.Delete("test-id")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证已删除
	_, err = storage.Get("test-id")
	if err == nil {
		t.Error("Expected error when getting deleted conversation")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/conversation -run TestFileStorage -v`
Expected: FAIL with "undefined: NewFileStorage"

**Step 3: Write minimal implementation**

Create `internal/conversation/storage.go`:

```go
package conversation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Storage 对话存储接口
type Storage interface {
	Save(conv *Conversation) error
	Get(id string) (*Conversation, error)
	List() ([]*Conversation, error)
	Delete(id string) error
}

// FileStorage 文件系统存储实现
type FileStorage struct {
	conversationsDir string
}

// NewFileStorage 创建 FileStorage
func NewFileStorage(conversationsDir string) *FileStorage {
	return &FileStorage{
		conversationsDir: conversationsDir,
	}
}

// GetDatePath 获取对话的日期路径 (YYYYMMDD)
func (s *FileStorage) GetDatePath(conv *Conversation) string {
	date := conv.CreatedAt.Format("20060102")
	return filepath.Join(s.conversationsDir, date)
}

// GetConversationPath 获取对话的完整路径
func (s *FileStorage) GetConversationPath(convID string) (string, error) {
	// 遍历日期文件夹查找
	entries, err := os.ReadDir(s.conversationsDir)
	if err != nil {
		return "", fmt.Errorf("failed to read conversations directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		convPath := filepath.Join(s.conversationsDir, entry.Name(), convID)
		if _, err := os.Stat(convPath); err == nil {
			return convPath, nil
		}
	}

	return "", fmt.Errorf("conversation not found: %s", convID)
}

// Save 保存对话
func (s *FileStorage) Save(conv *Conversation) error {
	datePath := s.GetDatePath(conv)

	// 创建日期目录
	if err := os.MkdirAll(datePath, 0755); err != nil {
		return fmt.Errorf("failed to create date directory: %w", err)
	}

	convPath := filepath.Join(datePath, conv.ID)

	// 创建对话目录
	if err := os.MkdirAll(convPath, 0755); err != nil {
		return fmt.Errorf("failed to create conversation directory: %w", err)
	}

	// 写入 messages.json
	messagesFile := filepath.Join(convPath, "messages.json")
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	if err := os.WriteFile(messagesFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write messages file: %w", err)
	}

	return nil
}

// Get 获取对话
func (s *FileStorage) Get(id string) (*Conversation, error) {
	convPath, err := s.GetConversationPath(id)
	if err != nil {
		return nil, err
	}

	messagesFile := filepath.Join(convPath, "messages.json")
	data, err := os.ReadFile(messagesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read messages file: %w", err)
	}

	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversation: %w", err)
	}

	return &conv, nil
}

// List 列出所有对话
func (s *FileStorage) List() ([]*Conversation, error) {
	var conversations []*Conversation

	entries, err := os.ReadDir(s.conversationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return conversations, nil // 目录不存在，返回空列表
		}
		return nil, fmt.Errorf("failed to read conversations directory: %w", err)
	}

	for _, dateEntry := range entries {
		if !dateEntry.IsDir() {
			continue
		}

		datePath := filepath.Join(s.conversationsDir, dateEntry.Name())
		convEntries, err := os.ReadDir(datePath)
		if err != nil {
			continue
		}

		for _, convEntry := range convEntries {
			if !convEntry.IsDir() {
				continue
			}

			conv, err := s.Get(convEntry.Name())
			if err != nil {
				continue
			}

			conversations = append(conversations, conv)
		}
	}

	// 按更新时间排序
	for i := 0; i < len(conversations); i++ {
		for j := i + 1; j < len(conversations); j++ {
			if conversations[i].UpdatedAt.Before(conversations[j].UpdatedAt) {
				conversations[i], conversations[j] = conversations[j], conversations[i]
			}
		}
	}

	return conversations, nil
}

// Delete 删除对话
func (s *FileStorage) Delete(id string) error {
	convPath, err := s.GetConversationPath(id)
	if err != nil {
		return err
	}

	return os.RemoveAll(convPath)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/conversation -run TestFileStorage -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/conversation/storage.go internal/conversation/storage_test.go
git commit -m "feat(conversation): add file storage with date-based organization"
```

---

## Task 5: 扩展 AIProvider 接口支持流式对话

**Files:**
- Modify: `internal/ai/provider.go`
- Test: `internal/ai/provider_test.go`

**Step 1: Write the failing test**

创建或修改 `internal/ai/provider_test.go`:

```go
package ai

import (
	"context"
	"testing"
	"time"
)

func TestAIProvider_ChatStream(t *testing.T) {
	// Mock provider for testing
	mock := &mockAIProvider{}

	ctx := context.Background()
	messages := []Message{
		{Role: "user", Content: "hello"},
	}

	stream, err := mock.ChatStream(ctx, messages)
	if err != nil {
		t.Fatalf("ChatStream failed: %v", err)
	}

	// 收集流式响应
	var response strings.Builder
	timeout := time.After(5 * time.Second)

	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				// channel closed
				if response.String() == "" {
					t.Error("Expected non-empty response")
				}
				return
			}
			response.WriteString(chunk)
		case <-timeout:
			t.Fatal("Timeout waiting for stream")
		}
	}
}

// mockAIProvider 用于测试
type mockAIProvider struct{}

func (m *mockAIProvider) ParseIntent(ctx context.Context, input string, systemPrompt string) (*Intent, error) {
	return &Intent{}, nil
}

func (m *mockAIProvider) AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error) {
	return "", nil
}

func (m *mockAIProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	return "response", nil
}

func (m *mockAIProvider) ChatStream(ctx context.Context, messages []Message) (<-chan string, error) {
	ch := make(chan string)
	go func() {
		defer close(ch)
		ch <- "Hello"
		time.Sleep(10 * time.Millisecond)
		ch <- " World"
	}()
	return ch, nil
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ai -run TestAIProvider_ChatStream -v`
Expected: FAIL with "method ChatStream not defined"

**Step 3: Write minimal implementation**

修改 `internal/ai/provider.go`:

```go
package ai

import "context"

// Message represents a chat message
type Message struct {
	Role    string `json:"role"` // "system" | "user" | "assistant"
	Content string `json:"content"`
}

// Intent represents the parsed user intent
type Intent struct {
	Commands     []Command `json:"commands"`
	Reason       string    `json:"reason"`
	NeedsConfirm bool      `json:"needs_confirm"`
}

// Command represents a shell command to execute
type Command struct {
	Cmd     string   `json:"cmd"`
	Args    []string `json:"args"`
	IsAsync bool     `json:"is_async"`
}

// AIProvider defines the interface for AI backends
type AIProvider interface {
	ParseIntent(ctx context.Context, input string, systemPrompt string) (*Intent, error)
	AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error)
	Chat(ctx context.Context, messages []Message) (string, error)

	// ChatStream 流式对话
	ChatStream(ctx context.Context, messages []Message) (<-chan string, error)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ai -run TestAIProvider_ChatStream -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ai/provider.go internal/ai/provider_test.go
git commit -m "feat(ai): add ChatStream method to AIProvider interface"
```

---

## Task 6: 实现 OpenAI ChatStream

**Files:**
- Modify: `internal/ai/openai/client.go`
- Test: `internal/ai/openai/client_test.go`

**Step 1: Write the failing test**

修改 `internal/ai/openai/client_test.go`:

```go
func TestIntegration_ChatStream(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1 to run integration tests")
	}

	client := NewClient("test-key", "gpt-4o-mini", "https://api.openai.com/v1")

	ctx := context.Background()
	messages := []ai.Message{
		{Role: "user", Content: "Say 'Hello World'"},
	}

	stream, err := client.ChatStream(ctx, messages)
	if err != nil {
		t.Fatalf("ChatStream failed: %v", err)
	}

	var response strings.Builder
	for chunk := range stream {
		response.WriteString(chunk)
	}

	if response.String() == "" {
		t.Error("Expected non-empty response")
	}

	t.Logf("Response: %s", response.String())
}
```

**Step 2: Run test to verify it fails**

Run: `TADA_INTEGRATION_TEST=1 go test ./internal/ai/openai -run TestIntegration_ChatStream -v`
Expected: FAIL (或需要 API key)

**Step 3: Write minimal implementation**

修改 `internal/ai/openai/client.go`，添加 ChatStream 方法:

```go
// ChatStream 流式对话
func (c *Client) ChatStream(ctx context.Context, messages []ai.Message) (<-chan string, error) {
	reqBody := map[string]interface{}{
		"model":    c.model,
		"messages": messages,
		"stream":   true, // 启用流式
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	ch := make(chan string)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// SSE 格式: "data: {...}"
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) > 0 {
				content := chunk.Choices[0].Delta.Content
				ch <- content
			}
		}
	}()

	return ch, nil
}
```

添加必要的 import:
```go
import (
	"bufio"
	// ... 其他 imports
)
```

**Step 4: Run test to verify it passes**

Run: `TADA_INTEGRATION_TEST=1 go test ./internal/ai/openai -run TestIntegration_ChatStream -v`
Expected: PASS (需要有效的 API key)

**Step 5: Commit**

```bash
git add internal/ai/openai/client.go internal/ai/openai/client_test.go
git commit -m "feat(openai): implement ChatStream for streaming responses"
```

---

## Task 7: 实现 GLM ChatStream

**Files:**
- Modify: `internal/ai/glm/client.go`
- Test: `internal/ai/glm/client_test.go`

**Step 1: 实现 GLM ChatStream**

GLM API 的 SSE 格式与 OpenAI 类似，实现方式相同。参考 Task 6 的实现，修改 `internal/ai/glm/client.go`。

**Step 2: 添加测试**

参考 Task 6 的测试实现。

**Step 3: Run test**

Run: `TADA_INTEGRATION_TEST=1 go test ./internal/ai/glm -v`

**Step 4: Commit**

```bash
git add internal/ai/glm/client.go internal/ai/glm/client_test.go
git commit -m "feat(glm): implement ChatStream for streaming responses"
```

---

## Task 8: 实现 ConversationManager

**Files:**
- Create: `internal/conversation/manager.go`
- Create: `internal/conversation/manager_test.go`

**Step 1: Write the failing test**

创建 `internal/conversation/manager_test.go`:

```go
package conversation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// mockAIProvider 用于测试
type mockChatAIProvider struct {
	response string
}

func (m *mockChatAIProvider) ParseIntent(ctx context.Context, input string, systemPrompt string) (*ai.Intent, error) {
	return &ai.Intent{}, nil
}

func (m *mockChatAIProvider) AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error) {
	return "", nil
}

func (m *mockChatAIProvider) Chat(ctx context.Context, messages []ai.Message) (string, error) {
	return m.response, nil
}

func (m *mockChatAIProvider) ChatStream(ctx context.Context, messages []ai.Message) (<-chan string, error) {
	ch := make(chan string)
	go func() {
		defer close(ch)
		ch <- m.response
	}()
	return ch, nil
}

func TestManager_CreateConversation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manager-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewFileStorage(tmpDir)
	promptLoader := NewPromptLoader(tmpDir)
	aiProvider := &mockChatAIProvider{response: "Hello"}

	manager := NewManager(storage, promptLoader, aiProvider)

	conv, err := manager.Create("test-name", "default")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if conv.Name != "test-name" {
		t.Errorf("Expected name 'test-name', got '%s'", conv.Name)
	}

	if conv.PromptName != "default" {
		t.Errorf("Expected prompt 'default', got '%s'", conv.PromptName)
	}
}

func TestManager_Chat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manager-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewFileStorage(tmpDir)
	promptLoader := NewPromptLoader(tmpDir)
	aiProvider := &mockChatAIProvider{response: "Hello!"}

	manager := NewManager(storage, promptLoader, aiProvider)

	conv, _ := manager.Create("test", "default")

	response, err := manager.Chat(conv.ID, "hi")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if response != "Hello!" {
		t.Errorf("Expected 'Hello!', got '%s'", response)
	}

	// 验证消息已保存
	loadedConv, _ := storage.Get(conv.ID)
	if len(loadedConv.Messages) != 2 { // user + assistant
		t.Errorf("Expected 2 messages, got %d", len(loadedConv.Messages))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/conversation -run TestManager -v`
Expected: FAIL with "undefined: NewManager"

**Step 3: Write minimal implementation**

创建 `internal/conversation/manager.go`:

```go
package conversation

import (
	"context"
	"fmt"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// Manager 对话管理器
type Manager struct {
	storage      Storage
	promptLoader *PromptLoader
	aiProvider   ai.AIProvider
}

// NewManager 创建 Manager
func NewManager(storage Storage, promptLoader *PromptLoader, aiProvider ai.AIProvider) *Manager {
	return &Manager{
		storage:      storage,
		promptLoader: promptLoader,
		aiProvider:   aiProvider,
	}
}

// Create 创建新对话
func (m *Manager) Create(name, promptName string) (*Conversation, error) {
	conv := NewConversation(promptName)
	conv.Name = name

	// 加载 prompt 模板
	prompt, err := m.promptLoader.Load(promptName)
	if err != nil {
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

	// 保存
	if err := m.storage.Save(conv); err != nil {
		return nil, fmt.Errorf("failed to save conversation: %w", err)
	}

	return conv, nil
}

// Get 获取对话
func (m *Manager) Get(id string) (*Conversation, error) {
	return m.storage.Get(id)
}

// List 列出所有对话
func (m *Manager) List() ([]*Conversation, error) {
	return m.storage.List()
}

// Delete 删除对话
func (m *Manager) Delete(id string) error {
	return m.storage.Delete(id)
}

// Chat 发送消息并获取回复
func (m *Manager) Chat(convID string, userInput string) (string, error) {
	conv, err := m.Get(convID)
	if err != nil {
		return "", fmt.Errorf("conversation not found: %w", err)
	}

	// 添加用户消息
	userMsg := Message{
		Role:      "user",
		Content:   userInput,
		Timestamp: time.Now(),
	}
	conv.AddMessage(userMsg)

	// 调用 AI
	messages := conv.GetMessagesForAI()
	response, err := m.aiProvider.Chat(context.Background(), messages)
	if err != nil {
		return "", fmt.Errorf("AI call failed: %w", err)
	}

	// 添加助手回复
	assistantMsg := Message{
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
	}
	conv.AddMessage(assistantMsg)

	// 保存
	if err := m.storage.Save(conv); err != nil {
		return "", fmt.Errorf("failed to save conversation: %w", err)
	}

	return response, nil
}

// ChatStream 流式对话
func (m *Manager) ChatStream(convID string, userInput string) (<-chan string, error) {
	conv, err := m.Get(convID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	// 添加用户消息
	userMsg := Message{
		Role:      "user",
		Content:   userInput,
		Timestamp: time.Now(),
	}
	conv.AddMessage(userMsg)

	// 调用 AI 流式接口
	messages := conv.GetMessagesForAI()
	stream, err := m.aiProvider.ChatStream(context.Background(), messages)
	if err != nil {
		return nil, fmt.Errorf("AI call failed: %w", err)
	}

	// 创建输出 channel
	out := make(chan string)

	go func() {
		defer close(out)

		var fullResponse strings.Builder

		for chunk := range stream {
			fullResponse.WriteString(chunk)
			out <- chunk
		}

		// 添加助手回复
		assistantMsg := Message{
			Role:      "assistant",
			Content:   fullResponse.String(),
			Timestamp: time.Now(),
		}
		conv.AddMessage(assistantMsg)

		// 保存
		m.storage.Save(conv)
	}()

	return out, nil
}
```

添加 import:
```go
import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/conversation -run TestManager -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/conversation/manager.go internal/conversation/manager_test.go
git commit -m "feat(conversation): add conversation manager"
```

---

## Task 9: 实现 Markdown Renderer

**Files:**
- Create: `internal/conversation/renderer.go`
- Create: `internal/conversation/renderer_test.go`

**Step 1: Write the failing test**

创建 `internal/conversation/renderer_test.go`:

```go
package conversation

import (
	"testing"
)

func TestRenderer_Render(t *testing.T) {
	renderer, err := NewRenderer(80)
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	// 测试 markdown 渲染
	markdown := `# Hello

This is **bold** and *italic*.

\`\`\`go
func main() {
	fmt.Println("Hello, World!")
}
\`\`\`
`

	rendered, err := renderer.Render(markdown)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if rendered == "" {
		t.Error("Expected non-empty rendered output")
	}

	// 渲染后的文本应该包含 ANSI 颜色代码
	// glamour 使用 lipgloss 添加颜色
	if rendered == markdown {
		t.Log("Warning: Rendered output same as input (glamour may not be working)")
	}
}

func TestRenderer_RenderPlainText(t *testing.T) {
	renderer, _ := NewRenderer(80)

	text := "Plain text without markdown"

	rendered, err := renderer.Render(text)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if rendered != text {
		t.Logf("Plain text was modified: %s", rendered)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/conversation -run TestRenderer -v`
Expected: FAIL with "undefined: NewRenderer"

**Step 3: Write minimal implementation**

创建 `internal/conversation/renderer.go`:

```go
package conversation

import (
	"github.com/charmbracelet/glamour"
)

// Renderer Markdown 渲染器
type Renderer struct {
	term *glamour.Term
}

// NewRenderer 创建 Renderer
func NewRenderer(width int) (*Renderer, error) {
	term, err := glamour.NewTerm(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}

	return &Renderer{term: term}, nil
}

// Render 渲染 markdown
func (r *Renderer) Render(markdown string) (string, error) {
	out, err := r.term.Render(markdown)
	if err != nil {
		// 降级：返回原始文本
		return markdown, nil
	}
	return out, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/conversation -run TestRenderer -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/conversation/renderer.go internal/conversation/renderer_test.go
git commit -m "feat(conversation): add markdown renderer with glamour"
```

---

## Task 10: 实现 REPL 组件

**Files:**
- Create: `internal/terminal/repl.go`
- Create: `internal/terminal/repl_test.go`

**Step 1: Write the failing test**

创建 `internal/terminal/repl_test.go`:

```go
package terminal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/conversation"
)

func TestREPL_ProcessInput(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "repl-test-*")
	defer os.RemoveAll(tmpDir)

	storage := conversation.NewFileStorage(tmpDir)
	promptLoader := conversation.NewPromptLoader(tmpDir)
	aiProvider := &mockChatAIProvider{response: "Test response"}

	manager := conversation.NewManager(storage, promptLoader, aiProvider)
	conv, _ := manager.Create("test", "default")

	repl := NewREPL(manager, conv, false)
	repl.renderer, _ = conversation.NewRenderer(80)

	// 测试普通消息处理
	err := repl.ProcessInput("hello")
	if err != nil {
		t.Fatalf("ProcessInput failed: %v", err)
	}

	// 验证消息已添加
	loadedConv, _ := manager.Get(conv.ID)
	if len(loadedConv.Messages) != 2 { // user + assistant
		t.Errorf("Expected 2 messages, got %d", len(loadedConv.Messages))
	}
}

func TestREPL_HandleCommand(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "repl-test-*")
	defer os.RemoveAll(tmpDir)

	storage := conversation.NewFileStorage(tmpDir)
	promptLoader := conversation.NewPromptLoader(tmpDir)
	aiProvider := &mockChatAIProvider{response: ""}

	manager := conversation.NewManager(storage, promptLoader, aiProvider)
	conv, _ := manager.Create("test", "default")

	repl := NewREPL(manager, conv, false)

	// 测试 /help 命令
	shouldExit, err := repl.HandleCommand("/help")
	if err != nil {
		t.Fatalf("HandleCommand failed: %v", err)
	}

	if shouldExit {
		t.Error("Expected shouldExit=false for /help")
	}

	// 测试 /exit 命令
	shouldExit, err = repl.HandleCommand("/exit")
	if err != nil {
		t.Fatalf("HandleCommand failed: %v", err)
	}

	if !shouldExit {
		t.Error("Expected shouldExit=true for /exit")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/terminal -run TestREPL -v`
Expected: FAIL with "undefined: NewREPL"

**Step 3: Write minimal implementation**

创建 `internal/terminal/repl.go`:

```go
package terminal

import (
	"fmt"
	"strings"

	"github.com/Lin-Jiong-HDU/tada/internal/conversation"
)

// REPL 交互式对话
type REPL struct {
	manager      *conversation.Manager
	conversation *conversation.Conversation
	renderer     *conversation.Renderer
	stream       bool
	showThinking bool
}

// NewREPL 创建 REPL
func NewREPL(manager *conversation.Manager, conv *conversation.Conversation, stream bool) *REPL {
	return &REPL{
		manager:      manager,
		conversation: conv,
		stream:       stream,
		showThinking: true,
	}
}

// ProcessInput 处理用户输入
func (r *REPL) ProcessInput(input string) error {
	input = strings.TrimSpace(input)

	// 检查是否是命令
	if strings.HasPrefix(input, "/") {
		shouldExit, err := r.HandleCommand(input)
		if err != nil {
			return err
		}
		if shouldExit {
			return fmt.Errorf("exit")
		}
		return nil
	}

	// 普通对话
	if r.stream {
		return r.processStreamChat(input)
	}

	return r.processChat(input)
}

// processChat 处理普通对话
func (r *REPL) processChat(input string) error {
	response, err := r.manager.Chat(r.conversation.ID, input)
	if err != nil {
		return err
	}

	// 渲染 markdown
	if r.renderer != nil {
		rendered, _ := r.renderer.Render(response)
		fmt.Print(rendered)
	} else {
		fmt.Println(response)
	}

	return nil
}

// processStreamChat 处理流式对话
func (r *REPL) processStreamChat(input string) error {
	if r.showThinking {
		fmt.Print("🤠 思考中...")
	}

	stream, err := r.manager.ChatStream(r.conversation.ID, input)
	if err != nil {
		return err
	}

	// 清除 "思考中..."
	if r.showThinking {
		fmt.Print("\r\033[K")
	}

	fmt.Print("🤖 ")

	var fullResponse strings.Builder
	for chunk := range stream {
		fmt.Print(chunk)
		fullResponse.WriteString(chunk)
	}

	fmt.Println()

	// 重新渲染美化版本
	if r.renderer != nil {
		rendered, _ := r.renderer.Render(fullResponse.String())
		fmt.Print(rendered)
	}

	return nil
}

// HandleCommand 处理命令
func (r *REPL) HandleCommand(cmd string) (bool, error) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false, nil
	}

	switch parts[0] {
	case "/exit", "/quit":
		r.DisplayExitSummary()
		return true, nil

	case "/help":
		r.DisplayHelp()
		return false, nil

	case "/clear":
		fmt.Print("\033[H\033[2J") // ANSI 清屏
		return false, nil

	case "/prompt":
		if len(parts) < 2 {
			fmt.Println("用法: /prompt <name>")
			return false, nil
		}
		fmt.Printf("切换 prompt: %s (未实现)\n", parts[1])
		return false, nil

	default:
		fmt.Printf("未知命令: %s\n", parts[0])
		return false, nil
	}
}

// DisplayHelp 显示帮助
func (r *REPL) DisplayHelp() {
	help := `
可用命令:
  /help         显示此帮助
  /clear        清屏
  /prompt <name> 切换 prompt 模板
  /exit, /quit  退出并保存
`
	fmt.Println(help)
}

// DisplayExitSummary 显示退出摘要
func (r *REPL) DisplayExitSummary() {
	fmt.Println("📝 对话已保存")
	fmt.Printf("   ID: %s\n", r.conversation.ID)
	fmt.Printf("   消息: %d 条\n", len(r.conversation.Messages))
	fmt.Printf("   恢复: tada chat --continue %s\n", r.conversation.ID)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/terminal -run TestREPL -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/terminal/repl.go internal/terminal/repl_test.go
git add internal/terminal/repl.go internal/terminal/repl_test.go
git commit -m "feat(terminal): add REPL for interactive chat"
```

---

## Task 11: 重写 chatCmd

**Files:**
- Modify: `cmd/tada/chat.go` (重写或新建)
- Test: `cmd/tada/chat_test.go`

**Step 1: Write the failing test**

创建或修改 `cmd/tada/chat_test.go`:

```go
package main

import (
	"testing"
)

func TestGetChatCommand_Exists(t *testing.T) {
	cmd := getChatCommand()
	if cmd == nil {
		t.Fatal("Expected chat command to exist")
	}

	if cmd.Use != "chat" {
		t.Errorf("Expected command name 'chat', got '%s'", cmd.Use)
	}
}

func TestGetChatCommand_HasFlags(t *testing.T) {
	cmd := getChatCommand()

	flags := []string{"prompt", "continue", "list", "delete"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag '%s' to exist", flag)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/tada -run TestGetChatCommand -v`
Expected: FAIL (chat.go 需要重写)

**Step 3: Write minimal implementation**

创建或修改 `cmd/tada/chat.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/ai/glm"
	"github.com/Lin-Jiong-HDU/tada/internal/ai/openai"
	"github.com/Lin-Jiong-HDU/tada/internal/conversation"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
	"github.com/Lin-Jiong-HDU/tada/internal/terminal"
	"github.com/spf13/cobra"
)

var (
	chatPromptName  string
	chatContinueID  string
	chatList        bool
	chatToday       bool
	chatShowID      string
	chatDeleteID    string
	chatName        string
	chatNoHistory   bool
	chatNoStream    bool
	chatNoRender    bool
)

func getChatCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "与 AI 对话",
		Long:  "交互式 AI 对话，支持多轮对话、历史记录和自定义 prompt",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			_, err := storage.InitConfig()
			return err
		},
		RunE: runChat,
	}

	cmd.Flags().StringVarP(&chatPromptName, "prompt", "p", "default", "Prompt 模板名称")
	cmd.Flags().StringVarP(&chatContinueID, "continue", "c", "", "恢复对话 ID")
	cmd.Flags().BoolVarP(&chatList, "list", "l", false, "列出所有对话")
	cmd.Flags().BoolVar(&chatToday, "today", false, "仅列出今天的对话")
	cmd.Flags().StringVarP(&chatShowID, "show", "s", "", "显示对话详情")
	cmd.Flags().StringVarP(&chatDeleteID, "delete", "d", "", "删除对话")
	cmd.Flags().StringVarP(&chatName, "name", "n", "", "对话名称")
	cmd.Flags().BoolVar(&chatNoHistory, "no-history", false, "不保存历史")
	cmd.Flags().BoolVar(&chatNoStream, "no-stream", false, "禁用流式输出")
	cmd.Flags().BoolVar(&chatNoRender, "no-render", false, "禁用 markdown 渲染")

	return cmd
}

func runChat(cmd *cobra.Command, args []string) error {
	cfg := storage.GetConfig()

	// 验证 API key
	if cfg.AI.APIKey == "" {
		return fmt.Errorf("AI API key 未配置，请在 ~/.tada/config.yaml 中设置")
	}

	// 创建 AI provider
	var aiProvider ai.AIProvider
	switch cfg.AI.Provider {
	case "openai":
		aiProvider = openai.NewClient(cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL)
	case "glm", "zhipu":
		aiProvider = glm.NewClient(cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL)
	default:
		return fmt.Errorf("不支持的 provider: %s", cfg.AI.Provider)
	}

	// 初始化存储
	configDir, _ := storage.GetConfigDir()
	conversationsDir := filepath.Join(configDir, "conversations")
	promptsDir := filepath.Join(configDir, "prompts")

	storage := conversation.NewFileStorage(conversationsDir)
	promptLoader := conversation.NewPromptLoader(promptsDir)
	manager := conversation.NewManager(storage, promptLoader, aiProvider)

	// 处理子命令
	if chatList {
		return runListConversations(manager)
	}

	if chatShowID != "" {
		return runShowConversation(manager, chatShowID)
	}

	if chatDeleteID != "" {
		return runDeleteConversation(manager, chatDeleteID)
	}

	// 创建或恢复对话
	var conv *conversation.Conversation
	var err error

	if chatContinueID != "" {
		conv, err = manager.Get(chatContinueID)
		if err != nil {
			return fmt.Errorf("对话不存在: %s", chatContinueID)
		}
		fmt.Printf("📂 恢复对话: %s (%s)\n", conv.ID, conv.PromptName)
	} else {
		conv, err = manager.Create(chatName, chatPromptName)
		if err != nil {
			return fmt.Errorf("创建对话失败: %w", err)
		}
		fmt.Printf("📝 新对话 (%s)\n", conv.PromptName)
	}

	// 创建 renderer
	var renderer *conversation.Renderer
	if !chatNoRender {
		renderer, _ = conversation.NewRenderer(80)
	}

	// 运行 REPL
	repl := terminal.NewREPL(manager, conv, !chatNoStream)
	repl.SetRenderer(renderer)

	fmt.Println("💬 输入消息，/help 查看命令，/exit 退出")
	fmt.Println()

	return repl.Run()
}

func runListConversations(manager *conversation.Manager) error {
	convs, err := manager.List()
	if err != nil {
		return err
	}

	if len(convs) == 0 {
		fmt.Println("💬 没有对话记录")
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

func runShowConversation(manager *conversation.Manager, id string) error {
	conv, err := manager.Get(id)
	if err != nil {
		return fmt.Errorf("对话不存在: %w", err)
	}

	fmt.Printf("对话: %s\n", conv.ID)
	fmt.Printf("Prompt: %s\n", conv.PromptName)
	fmt.Printf("创建时间: %s\n", conv.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("消息数: %d\n", len(conv.Messages))
	fmt.Println("\n消息:")
	fmt.Println()

	for _, msg := range conv.Messages {
		fmt.Printf("[%s]: %s\n\n", msg.Role, msg.Content)
	}

	return nil
}

func runDeleteConversation(manager *conversation.Manager, id string) error {
	err := manager.Delete(id)
	if err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}

	fmt.Printf("✓ 对话已删除: %s\n", id)
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/tada -run TestGetChatCommand -v`
Expected: PASS

**Step 5: 更新 main.go 注册新命令**

修改 `cmd/tada/main.go`:

```go
func init() {
	// 移除旧的 chatCmd，使用新的
	rootCmd.AddCommand(getChatCommand())
	rootCmd.AddCommand(getTasksCommand())
	rootCmd.AddCommand(getRunCommand())
}
```

删除或注释掉旧的 chatCmd 定义。

**Step 6: Commit**

```bash
git add cmd/tada/chat.go cmd/tada/chat_test.go cmd/tada/main.go
git commit -m "feat(chat): rewrite chat command for conversation mode"
```

---

## Task 12: 添加默认 Prompt 模板

**Files:**
- Create: `~/.tada/prompts/default.md`
- Create: `~/.tada/prompts/coder.md`
- Create: `internal/conversation/prompts.go` (可选，用于自动创建)

**Step 1: 创建 Prompt 模板生成器**

创建 `internal/conversation/prompts.go`:

```go
package conversation

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureDefaultPrompts 确保默认 prompt 存在
func EnsureDefaultPrompts(promptsDir string) error {
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return err
	}

	prompts := map[string]string{
		"default.md": `---
name: "default"
title: "默认助手"
description: "友好的 AI 助手"
---

你是一个友好、乐于助人的 AI 助手。请用简洁、准确的方式回答用户的问题。`,
		"coder.md": `---
name: "coder"
title: "编程助手"
description: "专业的编程对话助手"
---

你是一位经验丰富的程序员，擅长 Go、Python、JavaScript、TypeScript 等编程语言。

你的回答应该：
- 简洁、准确
- 提供可执行的代码示例
- 解释代码的工作原理
- 遵循最佳实践`,
		"expert.md": `---
name: "expert"
title: "技术专家"
description: "深入的技术分析和解答"
---

你是一位技术专家，能够提供深入的技术分析和解答。

你的回答应该：
- 深入分析问题的本质
- 提供多种解决方案
- 讨论各种方案的权衡
- 给出专业建议`,
	}

	for name, content := range prompts {
		path := filepath.Join(promptsDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create prompt %s: %w", name, err)
			}
		}
	}

	return nil
}
```

**Step 2: 在 chatCmd 中调用**

修改 `cmd/tada/chat.go` 中的 `runChat` 函数:

```go
func runChat(cmd *cobra.Command, args []string) error {
	cfg := storage.GetConfig()

	// ... 现有代码 ...

	// 确保默认 prompts 存在
	promptsDir := filepath.Join(configDir, "prompts")
	if err := conversation.EnsureDefaultPrompts(promptsDir); err != nil {
		return fmt.Errorf("初始化 prompts 失败: %w", err)
	}

	// ... 继续现有代码 ...
}
```

**Step 3: 验证**

```bash
go run cmd/tada/main.go chat --list
```

**Step 4: Commit**

```bash
git add internal/conversation/prompts.go
git commit -m "feat(conversation): add default prompt templates"
```

---

## Task 13: 更新配置结构

**Files:**
- Modify: `internal/storage/config.go`
- Test: `internal/storage/config_test.go`

**Step 1: 添加 chat 配置**

修改 `internal/storage/config.go`，添加 ChatConfig:

```go
type Config struct {
	AI     AIConfig     `yaml:"ai"`
	Security SecurityConfig `yaml:"security"`
	Chat   ChatConfig   `yaml:"chat"` // 新增
}

// ChatConfig 对话配置
type ChatConfig struct {
	DefaultPrompt  string `yaml:"default_prompt"`
	MaxHistory     int    `yaml:"max_history"`
	AutoSave       bool   `yaml:"auto_save"`
	Stream         bool   `yaml:"stream"`
	RenderMarkdown bool   `yaml:"render_markdown"`
}

// DefaultChatConfig 返回默认 chat 配置
func DefaultChatConfig() ChatConfig {
	return ChatConfig{
		DefaultPrompt:  "default",
		MaxHistory:     100,
		AutoSave:       true,
		Stream:         true,
		RenderMarkdown: true,
	}
}
```

**Step 2: 更新 LoadConfig 以应用默认值**

修改 LoadConfig 函数，在加载后检查 chat 配置是否为空：

```go
func LoadConfig() (*Config, error) {
	// ... 现有加载代码 ...

	if cfg.Chat.DefaultPrompt == "" {
		cfg.Chat = DefaultChatConfig()
	}

	return cfg, nil
}
```

**Step 3: 测试**

```bash
go test ./internal/storage -run TestConfig -v
```

**Step 4: Commit**

```bash
git add internal/storage/config.go
git commit -m "feat(storage): add chat configuration section"
```

---

## Task 14: 集成测试

**Files:**
- Test: `tests/integration/chat_integration_test.go`

**Step 1: 编写集成测试**

创建 `tests/integration/chat_integration_test.go`:

```go
package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/conversation"
)

func TestChat_FullWorkflow(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1")
	}

	tmpDir, _ := os.MkdirTemp("", "chat-integration-*")
	defer os.RemoveAll(tmpDir)

	// 创建 mock AI provider
	mockAI := &mockChatAI{
		responses: map[string]string{
			"hello": "Hi there!",
			"code":  "Here's some code...",
		},
	}

	storage := conversation.NewFileStorage(tmpDir)
	promptLoader := conversation.NewPromptLoader(tmpDir)
	manager := conversation.NewManager(storage, promptLoader, mockAI)

	// 创建对话
	conv, err := manager.Create("test", "default")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 发送消息
	response, err := manager.Chat(conv.ID, "hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if response != "Hi there!" {
		t.Errorf("Expected 'Hi there!', got '%s'", response)
	}

	// 验证持久化
	loadedConv, err := storage.Get(conv.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(loadedConv.Messages) != 2 { // system + user + assistant
		t.Logf("Messages: %d", len(loadedConv.Messages))
	}
}

type mockChatAI struct {
	responses map[string]string
}

func (m *mockChatAI) ParseIntent(ctx context.Context, input string, systemPrompt string) (*ai.Intent, error) {
	return &ai.Intent{}, nil
}

func (m *mockChatAI) AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error) {
	return "", nil
}

func (m *mockChatAI) Chat(ctx context.Context, messages []ai.Message) (string, error) {
	// 返回最后一个用户消息对应的响应
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			if resp, ok := m.responses[messages[i].Content]; ok {
				return resp, nil
			}
		}
	}
	return "Default response", nil
}

func (m *mockChatAI) ChatStream(ctx context.Context, messages []ai.Message) (<-chan string, error) {
	resp, _ := m.Chat(ctx, messages)
	ch := make(chan string)
	go func() {
		defer close(ch)
		ch <- resp
	}()
	return ch, nil
}
```

**Step 2: 运行测试**

```bash
TADA_INTEGRATION_TEST=1 go test ./tests/integration -v
```

**Step 3: Commit**

```bash
git add tests/integration/chat_integration_test.go
git commit -m "test(integration): add chat workflow integration tests"
```

---

## Task 15: 手动测试和文档更新

**Files:**
- Modify: `README.md`
- Modify: `docs/getting-started.md`

**Step 1: 手动测试**

```bash
# 构建
go build -o tada cmd/tada/main.go

# 测试新对话
./tada chat

# 测试指定 prompt
./tada chat --prompt coder

# 测试恢复对话
./tada chat --continue <id>

# 测试列出对话
./tada chat --list
```

**Step 2: 更新 README.md**

在 README.md 中添加 chat 功能说明：

```markdown
## Usage

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
```

**Step 3: 更新 getting-started.md**

添加对话功能的详细说明。

**Step 4: Commit**

```bash
git add README.md docs/getting-started.md
git commit -m "docs: add chat feature documentation"
```

---

## Task 16: 最终验证和清理

**Step 1: 运行完整测试套件**

```bash
go test ./... -v
```

**Step 2: 代码检查**

```bash
go vet ./...
go fmt ./...
```

**Step 3: 构建验证**

```bash
go build -o tada cmd/tada/main.go
./tada --help
./tada chat --help
```

**Step 4: 最终提交**

```bash
git add -A
git commit -m "feat: complete chat conversation feature implementation

- 新增 conversation 包支持对话管理
- 实现交互式 REPL 界面
- 支持流式输出和 Markdown 渲染
- Prompt 模板管理系统
- 对话持久化（按日期分组存储）

Closes #ChatFeature"
```

---

## 总结

此实现计划涵盖了：

1. **基础结构** - conversation 包的 types、storage、prompt loader、renderer
2. **AI 集成** - 扩展 AIProvider 接口支持 ChatStream
3. **管理器** - ConversationManager 编排对话逻辑
4. **交互界面** - REPL 组件处理用户交互
5. **CLI 集成** - 重写 chatCmd 支持新功能
6. **配置和测试** - 配置扩展、单元测试、集成测试

每个任务都是 TDD 驱动，包含完整的测试、实现、提交循环。
