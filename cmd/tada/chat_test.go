package main

import (
	"testing"
)

func TestGetChatCommand_Exists(t *testing.T) {
	cmd := getChatCommand()
	if cmd == nil {
		t.Fatal("Expected chat command to exist")
	}

	if cmd.Use != "chat" {
		t.Errorf("Expected command name 'chat', got '%s'", cmd.Use)
	}
}

func TestGetChatCommand_HasFlags(t *testing.T) {
	cmd := getChatCommand()

	flags := []string{"prompt", "continue", "list", "delete"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag '%s' to exist", flag)
		}
	}
}

func TestGetChatCommand_HasRequiredFlags(t *testing.T) {
	cmd := getChatCommand()

	// Check for flags that should exist
	requiredFlags := map[string]string{
		"prompt":   "p",
		"continue": "c",
		"list":     "l",
		"show":     "s",
		"delete":   "d",
		"name":     "n",
	}

	for flag, shorthand := range requiredFlags {
		f := cmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("Expected flag '%s' to exist", flag)
			continue
		}
		if f.Shorthand != shorthand {
			t.Errorf("Expected shorthand '%s' for flag '%s', got '%s'", shorthand, flag, f.Shorthand)
		}
	}
}
