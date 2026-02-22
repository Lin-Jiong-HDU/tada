package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
	"github.com/spf13/viper"
)

const (
	ConfigFileName = "config"
	ConfigFileType = "yaml"
	TadaDirName    = ".tada"
)

var config *Config

// Config holds the application configuration
type Config struct {
	AI       AIConfig                `mapstructure:"ai"`
	Security security.SecurityPolicy `mapstructure:"security"`
	Chat     ChatConfig              `mapstructure:"chat"`
}

// AIConfig holds AI-related configuration
type AIConfig struct {
	Provider  string `mapstructure:"provider"`
	APIKey    string `mapstructure:"api_key"`
	Model     string `mapstructure:"model"`
	BaseURL   string `mapstructure:"base_url"`
	Timeout   int    `mapstructure:"timeout"`
	MaxTokens int    `mapstructure:"max_tokens"`
}

// ChatConfig holds chat-related configuration
type ChatConfig struct {
	DefaultPrompt  string `mapstructure:"default_prompt"`
	MaxHistory     int    `mapstructure:"max_history"`
	AutoSave       bool   `mapstructure:"auto_save"`
	Stream         bool   `mapstructure:"stream"`
	RenderMarkdown bool   `mapstructure:"render_markdown"`
}

// DefaultChatConfig returns default chat configuration
func DefaultChatConfig() ChatConfig {
	return ChatConfig{
		DefaultPrompt:  "default",
		MaxHistory:     100,
		AutoSave:       true,
		Stream:         true,
		RenderMarkdown: true,
	}
}

// GetConfigDir returns the tada config directory path
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, TadaDirName), nil
}

// InitConfig initializes the configuration
func InitConfig() (*Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	// Create config directory if not exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	v := viper.New()
	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(configDir)

	// Set defaults
	v.SetDefault("ai.provider", "openai")
	v.SetDefault("ai.model", "gpt-4o")
	v.SetDefault("ai.base_url", "https://api.openai.com/v1")
	v.SetDefault("ai.timeout", 30)
	v.SetDefault("ai.max_tokens", 4096)

	// Security defaults
	v.SetDefault("security.command_level", "dangerous")
	v.SetDefault("security.allow_shell", true)
	v.SetDefault("security.allow_terminal_takeover", true)
	v.SetDefault("security.restricted_paths", []string{})
	v.SetDefault("security.readonly_paths", []string{})

	// Chat defaults
	v.SetDefault("chat.default_prompt", "default")
	v.SetDefault("chat.max_history", 100)
	v.SetDefault("chat.auto_save", true)
	v.SetDefault("chat.stream", true)
	v.SetDefault("chat.render_markdown", true)

	// Read config file (ignore if not exists)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config not found, will create with defaults
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	config = &cfg
	return config, nil
}

// GetConfig returns the loaded config
func GetConfig() *Config {
	return config
}

// SaveConfig saves the current config to file
func SaveConfig(cfg *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if not exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	v := viper.New()
	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(configDir)

	v.Set("ai.provider", cfg.AI.Provider)
	v.Set("ai.api_key", cfg.AI.APIKey)
	v.Set("ai.model", cfg.AI.Model)
	v.Set("ai.base_url", cfg.AI.BaseURL)
	v.Set("ai.timeout", cfg.AI.Timeout)
	v.Set("ai.max_tokens", cfg.AI.MaxTokens)

	// Save security config
	v.Set("security.command_level", cfg.Security.CommandLevel)
	v.Set("security.allow_shell", cfg.Security.AllowShell)
	v.Set("security.allow_terminal_takeover", cfg.Security.AllowTerminalTakeover)
	v.Set("security.restricted_paths", cfg.Security.RestrictedPaths)
	v.Set("security.readonly_paths", cfg.Security.ReadOnlyPaths)

	configPath := filepath.Join(configDir, ConfigFileName+"."+ConfigFileType)
	return v.WriteConfigAs(configPath)
}
