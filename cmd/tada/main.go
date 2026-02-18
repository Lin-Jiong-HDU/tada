package main

import (
	"fmt"
	"os"

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
		fmt.Printf("🪄 tada received: %s\n", input)
		fmt.Printf("Config: AI Provider = %s, Model = %s\n", cfg.AI.Provider, cfg.AI.Model)
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
