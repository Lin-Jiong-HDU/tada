package memory

import (
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

func TestManager_BuildContext(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	config.StoragePath = tmpDir

	provider := &MockAIProvider{}
	mgr, err := NewManager(config, provider, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	currentMessages := []ai.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
	}

	contextMsgs := mgr.BuildContext(currentMessages)

	// Should have system prompt + current messages
	if len(contextMsgs) < 3 {
		t.Errorf("Expected at least 3 messages (system + 2 current), got %d", len(contextMsgs))
	}

	// First message should be system prompt with memory context
	if contextMsgs[0].Role != "system" {
		t.Errorf("Expected first message to be system role, got %s", contextMsgs[0].Role)
	}
}

func TestManager_OnSessionEnd(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	config.StoragePath = tmpDir
	config.EntityThreshold = 2 // Lower threshold for testing

	provider := &MockAIProvider{
		response: `{"entities": ["Go"], "preferences": {}, "context": []}`,
	}

	mgr, err := NewManager(config, provider, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create a mock conversation implementing the interface
	conv := &mockConversation{
		id:    "test-conv",
		messages: []*mockMessage{
			{role: "user", content: "Tell me about Go"},
			{role: "assistant", content: "Go is..."},
		},
		updatedAt: time.Now(),
	}

	// This should not block (async internally)
	err = mgr.OnSessionEnd(conv)
	if err != nil {
		t.Fatalf("OnSessionEnd failed: %v", err)
	}

	// Give time for async processing
	time.Sleep(100 * time.Millisecond)

	summaries := mgr.shortTerm.GetSummaries()
	if len(summaries) == 0 {
		t.Error("Expected summary to be created")
	}
}

// mockConversation implements Conversation interface for testing
type mockConversation struct {
	id        string
	messages  []*mockMessage
	updatedAt time.Time
}

func (m *mockConversation) ID() string                       { return m.id }
func (m *mockConversation) GetMessages() []ConversationMessage { msgs := make([]ConversationMessage, len(m.messages)); for i, msg := range m.messages { msgs[i] = msg }; return msgs }
func (m *mockConversation) UpdatedAt() time.Time              { return m.updatedAt }

// mockMessage implements ConversationMessage interface for testing
type mockMessage struct {
	role    string
	content string
}

func (m *mockMessage) Role() string    { return m.role }
func (m *mockMessage) Content() string { return m.content }
