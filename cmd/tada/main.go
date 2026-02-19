package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/ai/glm"
	"github.com/Lin-Jiong-HDU/tada/internal/ai/openai"
	"github.com/Lin-Jiong-HDU/tada/internal/core"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
	"github.com/spf13/cobra"
)

var incognito bool

var rootCmd = &cobra.Command{
	Use:   "tada",
	Short: "Terminal AI assistant",
	Long:  "tada - A terminal AI assistant that understands natural language and executes commands",
	Args:  cobra.MinimumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		_, err := storage.InitConfig()
		if err != nil {
			return err
		}
		// Skip session init in incognito mode
		if !incognito {
			_, err = storage.InitSession()
		}
		return err
	},
	Run: func(cmd *cobra.Command, args []string) {
		cfg := storage.GetConfig()
		input := args[0]

		// Validate config
		if cfg.AI.APIKey == "" {
			fmt.Fprintf(os.Stderr, "❌ Error: AI API key not configured. Please set it in ~/.tada/config.yaml\n")
			fmt.Fprintf(os.Stderr, "Example:\n  ai:\n    api_key: sk-xxx\n")
			os.Exit(1)
		}

		// Initialize components - create AI provider based on config
		var aiProvider ai.AIProvider
		switch cfg.AI.Provider {
		case "openai":
			aiProvider = openai.NewClient(cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL)
		case "glm", "zhipu":
			aiProvider = glm.NewClient(cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL)
		default:
			fmt.Fprintf(os.Stderr, "❌ Error: unsupported provider '%s' (supported: openai, glm)\n", cfg.AI.Provider)
			os.Exit(1)
		}

		executor := core.NewExecutor(30 * time.Second)
		engine := core.NewEngine(aiProvider, executor)

		// Process request
		if err := engine.Process(context.Background(), input, ""); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&incognito, "incognito", "i", false, "Run in incognito mode (don't save history)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
