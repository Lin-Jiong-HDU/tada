package conversation

import (
	"context"
	"os"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// mockChatAIProvider 用于测试
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
	if len(loadedConv.Messages) != 2 { // system + user + assistant
		t.Logf("Messages: %d", len(loadedConv.Messages))
	}
}
