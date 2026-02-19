package glm

import (
	"os"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-key", "glm-5", "")

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.apiKey != "test-key" {
		t.Errorf("Expected apiKey 'test-key', got '%s'", client.apiKey)
	}
	if client.model != "glm-5" {
		t.Errorf("Expected model 'glm-5', got '%s'", client.model)
	}
	if client.baseURL != defaultAPIBaseURL {
		t.Errorf("Expected baseURL '%s', got '%s'", defaultAPIBaseURL, client.baseURL)
	}
}

func TestNewClient_CustomBaseURL(t *testing.T) {
	customURL := "https://custom.api.com"
	client := NewClient("test-key", "glm-5", customURL)

	if client.baseURL != customURL {
		t.Errorf("Expected baseURL '%s', got '%s'", customURL, client.baseURL)
	}
}

func TestParseIntentResponse(t *testing.T) {
	client := NewClient("key", "glm-5", "")
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
	client := NewClient("key", "glm-5", "")
	jsonResp := `{invalid json}`

	_, err := client.parseIntentResponse(jsonResp)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestIntegration_RealAPI(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1 to run integration tests")
	}

	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("GLM_API_KEY not set")
	}

	client := NewClient(apiKey, "glm-5", "")
	response, err := client.Chat(nil, []ai.Message{
		{Role: "user", Content: "Say 'Hello, tada!'"},
	})

	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}

	t.Logf("Response: %s", response)
}

func TestAI_Types(t *testing.T) {
	// Ensure ai.Message and ai.Intent types are compatible
	msg := ai.Message{
		Role:    "user",
		Content: "test",
	}

	if msg.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", msg.Role)
	}

	intent := ai.Intent{
		Commands: []ai.Command{
			{Cmd: "echo", Args: []string{"hello"}},
		},
		Reason:       "Test",
		NeedsConfirm: false,
	}

	if len(intent.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(intent.Commands))
	}
}
