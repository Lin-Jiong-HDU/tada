package storage

import (
	"os"
	"testing"
)

func TestInitSession(t *testing.T) {
	oldHome := os.Getenv("HOME")
	tmpDir, err := os.MkdirTemp("", "tada-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	session, err := InitSession()
	if err != nil {
		t.Fatalf("InitSession failed: %v", err)
	}

	if session.ID == "" {
		t.Error("Expected non-empty session ID")
	}
	if len(session.Messages) != 0 {
		t.Errorf("Expected empty messages, got %d", len(session.Messages))
	}
}

func TestAddMessage(t *testing.T) {
	oldHome := os.Getenv("HOME")
	tmpDir, err := os.MkdirTemp("", "tada-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	_, _ = InitSession()

	AddMessage("user", "hello")
	AddMessage("assistant", "hi there")

	session := GetCurrentSession()
	if len(session.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(session.Messages))
	}
	if session.Messages[0].Content != "hello" {
		t.Errorf("Expected 'hello', got '%s'", session.Messages[0].Content)
	}
}

func TestClearSession(t *testing.T) {
	oldHome := os.Getenv("HOME")
	tmpDir, err := os.MkdirTemp("", "tada-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	InitSession()
	AddMessage("user", "test")

	err = ClearSession()
	if err != nil {
		t.Fatalf("ClearSession failed: %v", err)
	}

	session := GetCurrentSession()
	if session != nil {
		t.Error("Expected nil session after clear")
	}
}
