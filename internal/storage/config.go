package storage

import (
	"fmt"
	"os"
	"path/filepath"

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
	AI AIConfig `mapstructure:"ai"`
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

	configPath := filepath.Join(configDir, ConfigFileName+"."+ConfigFileType)
	return v.WriteConfigAs(configPath)
}
