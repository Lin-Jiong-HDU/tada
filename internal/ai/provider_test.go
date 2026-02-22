package ai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCommand_IsAsync(t *testing.T) {
	cmd := Command{
		Cmd:     "rm",
		Args:    []string{"-rf", "/tmp/test"},
		IsAsync: true,
	}

	if !cmd.IsAsync {
		t.Error("Expected IsAsync to be true")
	}
}

func TestCommand_JSONMarshal(t *testing.T) {
	cmd := Command{
		Cmd:     "dd",
		Args:    []string{"if=/dev/zero", "of=file"},
		IsAsync: true,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify IsAsync is in JSON
	if !contains(string(data), "is_async") {
		t.Error("Expected is_async field in JSON")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

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
