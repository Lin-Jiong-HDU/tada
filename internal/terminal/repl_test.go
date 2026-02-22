package terminal

import (
	"context"
	"os"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/conversation"
)

// mockChatAIProvider 用于测试
type mockREPLAIProvider struct {
	response string
}

func (m *mockREPLAIProvider) ParseIntent(ctx context.Context, input string, systemPrompt string) (*ai.Intent, error) {
	return &ai.Intent{}, nil
}

func (m *mockREPLAIProvider) AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error) {
	return "", nil
}

func (m *mockREPLAIProvider) Chat(ctx context.Context, messages []ai.Message) (string, error) {
	return m.response, nil
}

func (m *mockREPLAIProvider) ChatStream(ctx context.Context, messages []ai.Message) (<-chan string, error) {
	ch := make(chan string)
	go func() {
		defer close(ch)
		ch <- m.response
	}()
	return ch, nil
}

func TestREPL_ProcessInput(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "repl-test-*")
	defer os.RemoveAll(tmpDir)

	storage := conversation.NewFileStorage(tmpDir)
	promptLoader := conversation.NewPromptLoader(tmpDir)
	aiProvider := &mockREPLAIProvider{response: "Test response"}

	manager := conversation.NewManager(storage, promptLoader, aiProvider)
	conv, _ := manager.Create("test", "default")

	repl := NewREPL(manager, conv, false)
	renderer, _ := conversation.NewRenderer(80)
	repl.SetRenderer(renderer)

	// 测试普通消息处理
	err := repl.ProcessInput("hello")
	if err != nil {
		t.Fatalf("ProcessInput failed: %v", err)
	}

	// 验证消息已添加
	loadedConv, _ := manager.Get(conv.ID)
	if len(loadedConv.Messages) < 2 { // system + user + assistant
		t.Errorf("Expected at least 2 messages, got %d", len(loadedConv.Messages))
	}
}

func TestREPL_HandleCommand(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "repl-test-*")
	defer os.RemoveAll(tmpDir)

	storage := conversation.NewFileStorage(tmpDir)
	promptLoader := conversation.NewPromptLoader(tmpDir)
	aiProvider := &mockREPLAIProvider{response: ""}

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
