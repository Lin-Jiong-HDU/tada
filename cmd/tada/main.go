package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai/openai"
	"github.com/Lin-Jiong-HDU/tada/internal/core"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tada",
	Short: "Terminal AI assistant",
	Long:  "tada - A terminal AI assistant that understands natural language and executes commands",
	Args:  cobra.MinimumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		_, err := storage.InitConfig()
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

		// Initialize components
		aiClient := openai.NewClient(cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL)
		executor := core.NewExecutor(30 * time.Second)
		engine := core.NewEngine(aiClient, executor)

		// Process request
		if err := engine.Process(context.Background(), input, ""); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
