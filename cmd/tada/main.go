package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/ai/glm"
	"github.com/Lin-Jiong-HDU/tada/internal/ai/openai"
	"github.com/Lin-Jiong-HDU/tada/internal/core"
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
	"github.com/spf13/cobra"
)

var incognito bool

var rootCmd = &cobra.Command{
	Use:   "tada",
	Short: "Terminal AI assistant",
	Long:  "tada - A terminal AI assistant that understands natural language and executes commands",
}

// quickCmd handles the single-shot command execution (backward compatibility)
var quickCmd = &cobra.Command{
	Use:    "quick [prompt]",
	Short:  "Quick command execution (deprecated - use bare argument instead)",
	Long:   "Quick command execution - understands natural language and executes commands",
	Args:   cobra.MinimumNArgs(1),
	Hidden: true, // Hide from help, kept for backward compatibility
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

		// Use security policy from config, or defaults if not set
		securityPolicy := &cfg.Security
		if securityPolicy.CommandLevel == "" {
			securityPolicy = security.DefaultPolicy()
		}

		engine := core.NewEngine(aiProvider, executor, securityPolicy)

		// Initialize queue with current session
		if !incognito {
			session := storage.GetCurrentSession()
			if session != nil {
				configDir, _ := storage.GetConfigDir()
				queueFile := filepath.Join(configDir, storage.SessionDirName, session.ID, "queue.json")
				q := queue.NewQueue(queueFile, session.ID)
				engine.SetQueue(q)
			}
		}

		// Process request
		if err := engine.Process(context.Background(), input, ""); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Add subcommands - use the new chat command from chat.go
	rootCmd.AddCommand(getChatCommand())
	rootCmd.AddCommand(getTasksCommand())
	rootCmd.AddCommand(getRunCommand())
	rootCmd.AddCommand(quickCmd) // Hidden command for backward compatibility

	quickCmd.PersistentFlags().BoolVarP(&incognito, "incognito", "i", false, "Run in incognito mode (don't save history)")
}

func main() {
	// If no args, run chat command with help
	if len(os.Args) == 1 {
		if err := rootCmd.Execute(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// For backward compatibility: if first arg is not a known command, treat as chat
	// Check exact match for commands (no path separators) to avoid conflicts with files/directories
	// Also exclude flags (starting with '-') to preserve help/flag behavior
	if len(os.Args) > 1 {
		arg := os.Args[1]
		// Only treat as command if it's an exact match without path separators and not a flag
		if arg != "chat" && arg != "tasks" && arg != "run" && arg != "help" &&
			!containsPathSeparator(arg) && !isFlag(arg) {
			// Prepend "chat" to args for backward compatibility
			args := append([]string{"chat"}, os.Args[1:]...)
			rootCmd.SetArgs(args)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// containsPathSeparator checks if the string contains path separators
func containsPathSeparator(s string) bool {
	for _, ch := range s {
		if ch == '/' || ch == '\\' {
			return true
		}
	}
	return false
}

// isFlag checks if the string is a CLI flag (starts with -)
func isFlag(s string) bool {
	return len(s) > 0 && s[0] == '-'
}
