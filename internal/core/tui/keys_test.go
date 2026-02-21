package tui

import (
	"testing"
)

func TestKeyMap_Help(t *testing.T) {
	km := defaultKeyMap()

	help := km.Help()
	helpText := help.String()
	if helpText == "" {
		t.Error("Expected help to be generated")
	}

	// Verify key bindings are present
	if !contains(helpText, "授权") || !contains(helpText, "拒绝") {
		t.Error("Expected help to contain authorize and reject actions")
	}
}

func TestKeyMap_Bindings(t *testing.T) {
	km := defaultKeyMap()

	// Check up/down bindings
	if km.Up.help == "" {
		t.Error("Expected Up binding")
	}
	if km.Down.help == "" {
		t.Error("Expected Down binding")
	}

	// Check action bindings
	if km.Authorize.help == "" {
		t.Error("Expected Authorize binding")
	}
	if km.Reject.help == "" {
		t.Error("Expected Reject binding")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
