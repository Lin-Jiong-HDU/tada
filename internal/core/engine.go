package core

import (
	"context"
	"fmt"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
)

// Engine orchestrates the AI workflow
type Engine struct {
	ai       ai.AIProvider
	executor *Executor
}

// NewEngine creates a new engine
func NewEngine(aiProvider ai.AIProvider, executor *Executor) *Engine {
	return &Engine{
		ai:       aiProvider,
		executor: executor,
	}
}

// Process handles a user request from input to output
func (e *Engine) Process(ctx context.Context, input string, systemPrompt string) error {
	// Add user message to session
	session := storage.GetCurrentSession()
	if session != nil {
		storage.AddMessage("user", input)
	}

	// Step 1: Parse intent
	fmt.Println("🧠 Thinking...")
	intent, err := e.ai.ParseIntent(ctx, input, systemPrompt)
	if err != nil {
		return fmt.Errorf("failed to parse intent: %w", err)
	}

	fmt.Printf("📝 Plan: %s\n", intent.Reason)

	// Step 2: Confirm if needed
	if intent.NeedsConfirm {
		// TODO: Implement TUI confirmation
		fmt.Println("⚠️  This command requires confirmation.")
		// For MVP, auto-confirm with warning
		fmt.Println("⚠️  Proceeding (confirmation will be added in TUI phase)...")
	}

	// Step 3: Execute commands
	for i, cmd := range intent.Commands {
		fmt.Printf("\n🔧 Executing [%d/%d]: %s %v\n", i+1, len(intent.Commands), cmd.Cmd, cmd.Args)

		result, err := e.executor.Execute(ctx, cmd)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		// Show output (truncated if too long)
		e.displayOutput(result.Output)

		// Step 4: Analyze result
		if result.Error != nil {
			fmt.Printf("📊 Command failed (exit code %d)\n", result.ExitCode)
		} else {
			analysis, err := e.ai.AnalyzeOutput(ctx, cmd.Cmd, result.Output)
			if err != nil {
				fmt.Printf("⚠️  Could not analyze output\n")
			} else {
				fmt.Printf("✅ %s\n", analysis)
			}
		}
	}

	// Add assistant response to session
	if session != nil {
		storage.AddMessage("assistant", intent.Reason)
	}

	return nil
}

// displayOutput shows command output with truncation
func (e *Engine) displayOutput(output string) {
	maxLines := 20
	lines := splitLines(output)

	if len(lines) > maxLines {
		fmt.Printf("📄 Output (%d lines, showing first %d):\n", len(lines), maxLines)
		for i := 0; i < maxLines; i++ {
			fmt.Printf("  %s\n", lines[i])
		}
		fmt.Printf("  ... (%d more lines)\n", len(lines)-maxLines)
	} else if output != "" {
		fmt.Printf("📄 Output:\n%s\n", output)
	}
}

func splitLines(s string) []string {
	lines := make([]string, 0)
	current := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
