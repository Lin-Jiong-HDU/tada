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
	Memory   MemoryConfig            `mapstructure:"memory"`
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

// StreamingConfig 流式输出配置
type StreamingConfig struct {
	// MaxDisplayLines 流式输出最大显示行数，0 表示不限制
	MaxDisplayLines int `mapstructure:"max_display_lines"`
}

// ChatConfig holds chat-related configuration
type ChatConfig struct {
	DefaultPrompt  string          `mapstructure:"default_prompt"`
	MaxHistory     int             `mapstructure:"max_history"`
	AutoSave       bool            `mapstructure:"auto_save"`
	Stream         bool            `mapstructure:"stream"`
	RenderMarkdown bool            `mapstructure:"render_markdown"`
	Streaming      StreamingConfig `mapstructure:"streaming"`
}

// MemoryConfig holds memory-related configuration
type MemoryConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	ShortTermMaxTokens int    `mapstructure:"short_term_max_tokens"`
	EntityThreshold    int    `mapstructure:"entity_threshold"`
	StoragePath        string `mapstructure:"storage_path"`
}

// DefaultChatConfig returns default chat configuration
func DefaultChatConfig() ChatConfig {
	return ChatConfig{
		DefaultPrompt:  "default",
		MaxHistory:     100,
		AutoSave:       true,
		Stream:         true,
		RenderMarkdown: true,
		Streaming: StreamingConfig{
			MaxDisplayLines: 10,
		},
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
	v.SetDefault("chat.streaming.max_display_lines", 10)

	// Memory defaults
	v.SetDefault("memory.enabled", true)
	v.SetDefault("memory.short_term_max_tokens", 4000)
	v.SetDefault("memory.entity_threshold", 5)
	v.SetDefault("memory.storage_path", "~/.tada/memory")

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

	// Validate streaming config
	if cfg.Chat.Streaming.MaxDisplayLines < 0 {
		cfg.Chat.Streaming.MaxDisplayLines = 10 // reset to default
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

	// Save chat config
	v.Set("chat.default_prompt", cfg.Chat.DefaultPrompt)
	v.Set("chat.max_history", cfg.Chat.MaxHistory)
	v.Set("chat.auto_save", cfg.Chat.AutoSave)
	v.Set("chat.stream", cfg.Chat.Stream)
	v.Set("chat.render_markdown", cfg.Chat.RenderMarkdown)
	v.Set("chat.streaming.max_display_lines", cfg.Chat.Streaming.MaxDisplayLines)

	// Save memory config
	v.Set("memory.enabled", cfg.Memory.Enabled)
	v.Set("memory.short_term_max_tokens", cfg.Memory.ShortTermMaxTokens)
	v.Set("memory.entity_threshold", cfg.Memory.EntityThreshold)
	v.Set("memory.storage_path", cfg.Memory.StoragePath)

	configPath := filepath.Join(configDir, ConfigFileName+"."+ConfigFileType)
	return v.WriteConfigAs(configPath)
}
