package integration

import (
	"context"
	"os"
	"path/filepath"

	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/conversation"
)

func TestChat_FullWorkflow(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1")
	}

	tmpDir, err := os.MkdirTemp("", "chat-integration-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create mock AI provider
	mockAI := &mockChatAI{
		responses: map[string]string{
			"hello": "Hi there!",
			"code":  "Here's some code...",
		},
	}

	storage := conversation.NewFileStorage(tmpDir)
	promptLoader := conversation.NewPromptLoader(tmpDir)
	manager := conversation.NewManager(storage, promptLoader, mockAI)

	// Create conversation
	conv, err := manager.Create("test", "default")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify initial state
	if conv.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", conv.Name)
	}

	// Send message
	response, err := manager.Chat(conv.ID, "hello")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if response != "Hi there!" {
		t.Errorf("Expected 'Hi there!', got '%s'", response)
	}

	// Verify persistence
	loadedConv, err := storage.Get(conv.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Check message count (system + user + assistant)
	if len(loadedConv.Messages) < 2 {
		t.Logf("Messages: %d", len(loadedConv.Messages))
	}
}

func TestChat_StreamWorkflow(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1")
	}

	tmpDir, err := os.MkdirTemp("", "chat-stream-integration-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create mock AI provider
	mockAI := &mockChatAI{
		responses: map[string]string{
			"stream": "Stream response",
		},
	}

	storage := conversation.NewFileStorage(tmpDir)
	promptLoader := conversation.NewPromptLoader(tmpDir)
	manager := conversation.NewManager(storage, promptLoader, mockAI)

	// Create conversation
	conv, err := manager.Create("stream-test", "default")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test streaming chat
	stream, err := manager.ChatStream(conv.ID, "stream")
	if err != nil {
		t.Fatalf("ChatStream failed: %v", err)
	}

	var fullResponse string
	for chunk := range stream {
		fullResponse += chunk
	}

	if fullResponse != "Stream response" {
		t.Errorf("Expected 'Stream response', got '%s'", fullResponse)
	}
}

func TestChat_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chat-persistence-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mockAI := &mockChatAI{
		responses: map[string]string{
			"test": "Test response",
		},
	}

	conversationsDir := filepath.Join(tmpDir, "conversations")
	storage := conversation.NewFileStorage(conversationsDir)
	promptLoader := conversation.NewPromptLoader(tmpDir)
	manager := conversation.NewManager(storage, promptLoader, mockAI)

	// Create conversation
	conv1, err := manager.Create("persistence-test", "default")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Send a message
	_, err = manager.Chat(conv1.ID, "test")
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	// List conversations
	list, err := manager.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) == 0 {
		t.Error("Expected at least 1 conversation")
	}

	// Verify the conversation is in the list
	found := false
	for _, c := range list {
		if c.ID == conv1.ID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Created conversation not found in list")
	}
}

func TestChat_PromptLoading(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chat-prompt-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test prompt file
	promptContent := `---
name: "test-prompt"
title: "Test"
description: "Test prompt"
---
You are a test assistant.`

	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	promptFile := filepath.Join(promptsDir, "test-prompt.md")
	if err := os.WriteFile(promptFile, []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load the prompt
	loader := conversation.NewPromptLoader(promptsDir)
	prompt, err := loader.Load("test-prompt")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if prompt.Name != "test-prompt" {
		t.Errorf("Expected name 'test-prompt', got '%s'", prompt.Name)
	}

	if prompt.SystemPrompt != "You are a test assistant." {
		t.Errorf("Expected system prompt 'You are a test assistant.', got '%s'", prompt.SystemPrompt)
	}
}

// mockChatAI is a mock AI provider for testing
type mockChatAI struct {
	responses map[string]string
}

func (m *mockChatAI) ParseIntent(ctx context.Context, input string, systemPrompt string) (*ai.Intent, error) {
	return &ai.Intent{}, nil
}

func (m *mockChatAI) AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error) {
	return "", nil
}

func (m *mockChatAI) Chat(ctx context.Context, messages []ai.Message) (string, error) {
	// Return the response for the last user message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			if resp, ok := m.responses[messages[i].Content]; ok {
				return resp, nil
			}
		}
	}
	return "Default response", nil
}

func (m *mockChatAI) ChatStream(ctx context.Context, messages []ai.Message) (<-chan string, error) {
	resp, _ := m.Chat(ctx, messages)
	ch := make(chan string)
	go func() {
		defer close(ch)
		ch <- resp
	}()
	return ch, nil
}
