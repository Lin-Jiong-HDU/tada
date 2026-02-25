package conversation

import (
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/memory"
)

// MemoryAdapter adapts conversation.Conversation to memory.Conversation interface
type MemoryAdapter struct {
	conv *Conversation
}

// NewMemoryAdapter creates a new memory adapter
func NewMemoryAdapter(conv *Conversation) *MemoryAdapter {
	return &MemoryAdapter{conv: conv}
}

// ID returns the conversation ID
func (a *MemoryAdapter) ID() string {
	return a.conv.ID
}

// GetMessages returns the conversation messages
func (a *MemoryAdapter) GetMessages() []memory.ConversationMessage {
	msgs := make([]memory.ConversationMessage, len(a.conv.Messages))
	for i, msg := range a.conv.Messages {
		msgs[i] = memoryMessage{msg: &msg}
	}
	return msgs
}

// UpdatedAt returns the last update time
func (a *MemoryAdapter) UpdatedAt() time.Time {
	return a.conv.UpdatedAt
}

// memoryMessage adapts conversation.Message to memory.ConversationMessage interface
type memoryMessage struct {
	msg *Message
}

// Role returns the message role
func (m memoryMessage) Role() string {
	return m.msg.Role
}

// Content returns the message content
func (m memoryMessage) Content() string {
	return m.msg.Content
}
