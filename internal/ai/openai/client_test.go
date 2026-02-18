package openai

import (
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-key", "gpt-4", "https://api.openai.com/v1")

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.apiKey != "test-key" {
		t.Errorf("Expected apiKey 'test-key', got '%s'", client.apiKey)
	}
	if client.model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", client.model)
	}
}

func TestParseIntentResponse(t *testing.T) {
	client := NewClient("key", "model", "url")
	jsonResp := `{
		"commands": [{"cmd": "mkdir", "args": ["docs"]}],
		"reason": "Creating directory",
		"needs_confirm": false
	}`

	intent, err := client.parseIntentResponse(jsonResp)
	if err != nil {
		t.Fatalf("parseIntentResponse failed: %v", err)
	}

	if len(intent.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(intent.Commands))
	}
	if intent.Commands[0].Cmd != "mkdir" {
		t.Errorf("Expected cmd 'mkdir', got '%s'", intent.Commands[0].Cmd)
	}
	if intent.Reason != "Creating directory" {
		t.Errorf("Expected reason 'Creating directory', got '%s'", intent.Reason)
	}
}

func TestParseIntentResponse_InvalidJSON(t *testing.T) {
	client := NewClient("key", "model", "url")
	jsonResp := `{invalid json}`

	_, err := client.parseIntentResponse(jsonResp)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

// Test integration with AI types
func TestAI_Types(t *testing.T) {
	// This test verifies that the ai package types are properly accessible
	var intent ai.Intent
	intent.Commands = []ai.Command{{Cmd: "test"}}

	var msg ai.Message
	msg.Role = "user"
	msg.Content = "hello"

	_ = intent
	_ = msg
}
