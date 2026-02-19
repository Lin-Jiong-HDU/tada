package integration

import (
	"os"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/core"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
)

func TestSecurityIntegration_ConfigLoading(t *testing.T) {
	oldHome := os.Getenv("HOME")
	tmpDir, err := os.MkdirTemp("", "tada-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create config with security settings
	cfg, err := storage.InitConfig()
	if err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	// Verify defaults
	if cfg.Security.CommandLevel != security.ConfirmDangerous {
		t.Errorf("Expected default command_level 'dangerous', got '%s'", cfg.Security.CommandLevel)
	}
}

func TestSecurityIntegration_EngineCreation(t *testing.T) {
	policy := security.DefaultPolicy()
	engine := core.NewEngine(nil, core.NewExecutor(0), policy)

	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}
}

func TestSecurityIntegration_FullWorkflow(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1")
	}

	// This would test the full security workflow
	// Implementation depends on having a working AI provider
}
