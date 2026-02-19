package core

import (
	"os"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

func TestEngine_SecurityCheck(t *testing.T) {
	policy := &security.SecurityPolicy{
		CommandLevel:    security.ConfirmDangerous,
		RestrictedPaths: []string{"/etc"},
		ReadOnlyPaths:   []string{"~/.ssh"},
		AllowShell:      true,
	}

	engine := NewEngine(nil, NewExecutor(30*time.Second), policy)

	// Test that security controller is set
	if engine.securityController == nil {
		t.Error("Security controller should be initialized")
	}
}

func TestEngine_SecurityCheckIntegration(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	// This test requires a mock AI provider
	// For now, just verify the structure
	policy := security.DefaultPolicy()
	engine := NewEngine(nil, NewExecutor(30*time.Second), policy)

	if engine.securityController == nil {
		t.Error("Expected security controller to be initialized")
	}
}
