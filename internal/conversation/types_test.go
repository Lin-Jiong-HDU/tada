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
