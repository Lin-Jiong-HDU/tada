package security

import (
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

func TestSecurityController_CheckCommand(t *testing.T) {
	policy := &SecurityPolicy{
		CommandLevel:    ConfirmDangerous,
		RestrictedPaths: []string{"/etc"},
		ReadOnlyPaths:   []string{"~/.ssh"},
		AllowShell:      true,
	}
	controller := NewSecurityController(policy)

	t.Run("safe command allowed", func(t *testing.T) {
		result, err := controller.CheckCommand(ai.Command{Cmd: "ls", Args: []string{}})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("Expected safe command to be allowed")
		}
	})

	t.Run("dangerous command requires auth", func(t *testing.T) {
		result, err := controller.CheckCommand(ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !result.RequiresAuth {
			t.Error("Expected dangerous command to require auth")
		}
	})

	t.Run("restricted path rejected", func(t *testing.T) {
		result, err := controller.CheckCommand(ai.Command{Cmd: "cat", Args: []string{"/etc/passwd"}})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("Expected restricted path access to be rejected")
		}
	})
}
