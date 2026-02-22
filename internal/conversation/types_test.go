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
