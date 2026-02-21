package ai

import (
	"encoding/json"
	"testing"
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
