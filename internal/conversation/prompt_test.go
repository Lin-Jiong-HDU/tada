package conversation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPromptLoader_Load(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prompt-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试 prompt 文件
	promptContent := `---
name: "test"
title: "Test Prompt"
description: "A test prompt"
---

You are a test assistant.`

	promptFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(promptFile, []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewPromptLoader(tmpDir)
	prompt, err := loader.Load("test")

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if prompt.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", prompt.Name)
	}

	if prompt.Title != "Test Prompt" {
		t.Errorf("Expected title 'Test Prompt', got '%s'", prompt.Title)
	}

	if prompt.SystemPrompt != "You are a test assistant." {
		t.Errorf("Expected system prompt 'You are a test assistant.', got '%s'", prompt.SystemPrompt)
	}
}

func TestPromptLoader_List(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prompt-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建多个 prompt 文件
	prompt1 := `---
name: "default"
---
Default prompt`
	prompt2 := `---
name: "coder"
---
Coder prompt`

	os.WriteFile(filepath.Join(tmpDir, "default.md"), []byte(prompt1), 0644)
	os.WriteFile(filepath.Join(tmpDir, "coder.md"), []byte(prompt2), 0644)

	loader := NewPromptLoader(tmpDir)
	prompts, err := loader.List()

	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(prompts) != 2 {
		t.Errorf("Expected 2 prompts, got %d", len(prompts))
	}
}
