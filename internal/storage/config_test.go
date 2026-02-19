package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir failed: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home dir: %v", err)
	}

	expected := filepath.Join(home, TadaDirName)
	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}
}

func TestInitConfig(t *testing.T) {
	// Use temp directory for testing
	oldHome := os.Getenv("HOME")
	tmpDir, err := os.MkdirTemp("", "tada-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cfg, err := InitConfig()
	if err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	// Check defaults
	if cfg.AI.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", cfg.AI.Provider)
	}
	if cfg.AI.Model != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o', got '%s'", cfg.AI.Model)
	}
}

func TestSaveConfig(t *testing.T) {
	oldHome := os.Getenv("HOME")
	tmpDir, err := os.MkdirTemp("", "tada-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cfg := &Config{
		AI: AIConfig{
			Provider:  "test",
			APIKey:    "test-key",
			Model:     "test-model",
			BaseURL:   "https://test.com",
			Timeout:   60,
			MaxTokens: 2048,
		},
	}

	err = SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file exists
	configDir, _ := GetConfigDir()
	configPath := filepath.Join(configDir, ConfigFileName+"."+ConfigFileType)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file was not created")
	}
}
