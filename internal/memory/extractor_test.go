package memory

import (
	"context"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// MockAIProvider for testing
type MockAIProvider struct {
	response string
}

func (m *MockAIProvider) Chat(ctx context.Context, messages []ai.Message) (string, error) {
	return m.response, nil
}

func (m *MockAIProvider) ChatStream(ctx context.Context, messages []ai.Message) (<-chan string, error) {
	return nil, nil
}

func (m *MockAIProvider) ParseIntent(ctx context.Context, input string, systemPrompt string) (*ai.Intent, error) {
	return nil, nil
}

func (m *MockAIProvider) AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error) {
	return "", nil
}

func TestExtractor_ExtractFromSummary(t *testing.T) {
	mockResponse := `{
        "entities": ["Go", "React", "neovim"],
        "preferences": {
            "editor": "neovim",
            "timezone": "Asia/Shanghai"
        },
        "context": ["Working on tada CLI tool", "Interested in memory systems"]
    }`

	provider := &MockAIProvider{response: mockResponse}
	extractor := NewExtractor(provider, nil)

	result, err := extractor.ExtractFromSummary(context.Background(), "User discussed Go memory management and React state")
	if err != nil {
		t.Fatalf("ExtractFromSummary failed: %v", err)
	}

	if len(result.Entities) != 3 {
		t.Errorf("Expected 3 entities, got %d", len(result.Entities))
	}

	if result.Preferences["editor"] != "neovim" {
		t.Errorf("Expected editor preference 'neovim', got '%s'", result.Preferences["editor"])
	}
}
