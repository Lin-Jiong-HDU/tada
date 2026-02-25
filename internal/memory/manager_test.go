package memory

import (
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/conversation"
)

func TestManager_BuildContext(t *testing.T) {
	tmpDir := t.TempDir()
	config := DefaultConfig()
	config.StoragePath = tmpDir

	provider := &MockAIProvider{}
	mgr, err := NewManager(config, provider)
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

	mgr, err := NewManager(config, provider)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	conv := &conversation.Conversation{
		ID:   "test-conv",
		Name: "Test Conversation",
		Messages: []conversation.Message{
			{Role: "user", Content: "Tell me about Go", Timestamp: time.Now()},
			{Role: "assistant", Content: "Go is...", Timestamp: time.Now()},
		},
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
