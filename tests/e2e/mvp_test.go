package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai/openai"
	"github.com/Lin-Jiong-HDU/tada/internal/core"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
)

func TestMVP_FullWorkflow(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1 to run E2E tests")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	// Setup
	oldHome := os.Getenv("HOME")
	tmpDir, err := os.MkdirTemp("", "tada-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Initialize
	_, err = storage.InitConfig()
	if err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	_, err = storage.InitSession()
	if err != nil {
		t.Fatalf("InitSession failed: %v", err)
	}

	// Create engine
	aiClient := openai.NewClient(apiKey, "gpt-4o-mini", "https://api.openai.com/v1")
	executor := core.NewExecutor(30 * time.Second)
	engine := core.NewEngine(aiClient, executor)

	// Test simple command
	err = engine.Process(context.Background(), "say hello to the world", "")
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify session was saved
	session := storage.GetCurrentSession()
	if len(session.Messages) < 2 {
		t.Errorf("Expected at least 2 messages in session, got %d", len(session.Messages))
	}

	t.Log("E2E test passed!")
}
