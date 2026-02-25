package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/ai/glm"
	"github.com/Lin-Jiong-HDU/tada/internal/ai/openai"
	"github.com/Lin-Jiong-HDU/tada/internal/memory"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
)

// TestMemoryFullFlow tests the complete memory workflow
// Run with: TADA_INTEGRATION_TEST=1 go test ./tests/integration -run TestMemoryFullFlow -v
func TestMemoryFullFlow(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test. Set TADA_INTEGRATION_TEST=1 to run.")
	}

	// Setup
	tmpDir := t.TempDir()

	// Initialize memory prompts
	memoryPromptsDir := filepath.Join(tmpDir, "prompts", "memory")
	if err := memory.EnsureDefaultPrompts(memoryPromptsDir); err != nil {
		t.Fatalf("Failed to initialize memory prompts: %v", err)
	}

	config := &memory.Config{
		Enabled:            true,
		ShortTermMaxTokens: 1000,
		EntityThreshold:    2,
		StoragePath:        tmpDir,
	}

	// Get real AI config from environment
	appConfig := storage.GetConfig()
	if appConfig == nil {
		t.Skip("No valid config found, skipping integration test")
	}

	if appConfig.AI.APIKey == "" {
		t.Skip("AI API key not configured, skipping integration test")
	}

	// Create AI provider
	var provider ai.AIProvider
	switch appConfig.AI.Provider {
	case "openai":
		provider = openai.NewClient(appConfig.AI.APIKey, appConfig.AI.Model, appConfig.AI.BaseURL)
	case "glm", "zhipu":
		provider = glm.NewClient(appConfig.AI.APIKey, appConfig.AI.Model, appConfig.AI.BaseURL)
	default:
		t.Skipf("Unsupported provider: %s", appConfig.AI.Provider)
	}

	memoryPromptLoader := memory.NewPromptLoader(memoryPromptsDir)
	mgr, err := memory.NewManager(config, provider, memoryPromptLoader)
	if err != nil {
		t.Fatalf("Failed to create memory manager: %v", err)
	}

	if mgr == nil {
		t.Fatal("Memory manager is nil")
	}

	// Create a test conversation adapter
	now := time.Now()
	conv := &testConversation{
		id: "test-integration-conv",
		messages: []*testMessage{
			{role: "user", content: "I'm working on a Go project called tada"},
			{role: "assistant", content: "That sounds interesting! Tell me more about tada."},
			{role: "user", content: "It's a CLI tool written in Go that helps with terminal tasks"},
		},
		updatedAt: now,
	}

	// Process session end
	err = mgr.OnSessionEnd(conv)
	if err != nil {
		t.Fatalf("OnSessionEnd failed: %v", err)
	}

	// Wait for async processing
	time.Sleep(3 * time.Second)

	// Verify summary was created
	summaries := filepath.Join(tmpDir, "summaries.json")
	if _, err := os.Stat(summaries); os.IsNotExist(err) {
		t.Error("summaries.json was not created")
	} else {
		t.Logf("summaries.json created successfully")
	}

	// Verify entities were tracked
	entities := filepath.Join(tmpDir, "entities.json")
	if _, err := os.Stat(entities); os.IsNotExist(err) {
		t.Error("entities.json was not created")
	} else {
		t.Logf("entities.json created successfully")
	}

	// Verify profile was created
	profile := filepath.Join(tmpDir, "user_profile.json")
	if _, err := os.Stat(profile); os.IsNotExist(err) {
		t.Error("user_profile.json was not created")
	} else {
		t.Logf("user_profile.json created successfully")
	}

	// Test context building
	testMessages := []ai.Message{
		{Role: "user", Content: "What should I work on next?"},
	}

	contextMsgs := mgr.BuildContext(testMessages)

	// Should have system prompt + user message
	if len(contextMsgs) < 2 {
		t.Errorf("Expected at least 2 messages, got %d", len(contextMsgs))
	}

	if contextMsgs[0].Role != "system" {
		t.Errorf("Expected first message to be system, got %s", contextMsgs[0].Role)
	}

	t.Log("Integration test passed!")
}

// testConversation implements memory.Conversation interface for testing
type testConversation struct {
	id        string
	messages  []*testMessage
	updatedAt time.Time
}

func (t *testConversation) ID() string { return t.id }
func (t *testConversation) GetMessages() []memory.ConversationMessage {
	msgs := make([]memory.ConversationMessage, len(t.messages))
	for i, msg := range t.messages {
		msgs[i] = msg
	}
	return msgs
}
func (t *testConversation) UpdatedAt() time.Time { return t.updatedAt }

// testMessage implements memory.ConversationMessage interface for testing
type testMessage struct {
	role    string
	content string
}

func (t *testMessage) Role() string    { return t.role }
func (t *testMessage) Content() string { return t.content }
