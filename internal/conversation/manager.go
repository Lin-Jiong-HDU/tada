package conversation

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	// 使用局部变量避免在 goroutine 中引用可能变化的变量
	id := conv.ID

	go func() {
		defer close(out)

		var fullResponse strings.Builder

		for chunk := range stream {
			fullResponse.WriteString(chunk)
			out <- chunk
		}

		// 重新加载对话以避免竞态条件
		reloadedConv, err := m.Get(id)
		if err != nil {
			return // 对话不存在，无法保存
		}

		// 添加助手回复
		assistantMsg := Message{
			Role:      "assistant",
			Content:   fullResponse.String(),
			Timestamp: time.Now(),
		}
		reloadedConv.AddMessage(assistantMsg)

		// 保存
		_ = m.storage.Save(reloadedConv) // 保存失败时至少已发送到 channel
	}()

	return out, nil
}
