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
