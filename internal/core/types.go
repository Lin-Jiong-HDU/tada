package core

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
