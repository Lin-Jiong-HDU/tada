package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
	"github.com/Lin-Jiong-HDU/tada/internal/terminal"
)

// Engine orchestrates the AI workflow
type Engine struct {
	ai                 ai.AIProvider
	executor           *Executor
	securityController *security.SecurityController
	queue              *queue.Manager
}

// NewEngine creates a new engine
func NewEngine(aiProvider ai.AIProvider, executor *Executor, securityPolicy *security.SecurityPolicy) *Engine {
	return &Engine{
		ai:                 aiProvider,
		executor:           executor,
		securityController: security.NewSecurityController(securityPolicy),
	}
}

// SetQueue sets the task queue for async commands
func (e *Engine) SetQueue(q *queue.Manager) {
	e.queue = q
}

// ParseAsyncSyntax checks if the input ends with & for async execution
func ParseAsyncSyntax(input string) bool {
	trimmed := strings.TrimSpace(input)
	return strings.HasSuffix(trimmed, "&")
}

// StripAsyncSyntax removes trailing & from input
func StripAsyncSyntax(input string) string {
	trimmed := strings.TrimSpace(input)
	if strings.HasSuffix(trimmed, "&") {
		return strings.TrimSpace(trimmed[:len(trimmed)-1])
	}
	return trimmed
}

// Process handles a user request from input to output
func (e *Engine) Process(ctx context.Context, input string, systemPrompt string) error {
	// Check for async syntax
	isAsync := ParseAsyncSyntax(input)
	if isAsync {
		input = StripAsyncSyntax(input)
		// Validate that we have an actual command
		if input == "" {
			return fmt.Errorf("async marker '&' requires a command")
		}
	}

	// Add user message to session (use original input for history)
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

	// Mark all commands as async if & was used
	if isAsync {
		for i := range intent.Commands {
			intent.Commands[i].IsAsync = true
		}
	}

	// Only show plan for synchronous commands
	if !isAsync {
		fmt.Printf("📝 Plan: %s\n", intent.Reason)
	}

	// Step 2: Execute commands (with security check)
	for i, cmd := range intent.Commands {
		// Security check before execution
		result, err := e.securityController.CheckCommand(cmd)
		if err != nil {
			return fmt.Errorf("security check failed: %w", err)
		}

		if !result.Allowed {
			fmt.Printf("🚫 拒绝执行: %s\n", result.Reason)
			continue
		}

		// Handle async commands - always queue them
		if cmd.IsAsync {
			if e.queue != nil {
				task, err := e.queue.AddTask(cmd, result)
				if err != nil {
					return fmt.Errorf("failed to queue task: %w", err)
				}
				fmt.Printf("📋 命令已加入队列 (ID: %s)\n", task.ID)
				fmt.Printf("   使用 'tada tasks' 查看并授权\n")
				continue
			}
			// Fall through to sync execution if no queue available
		}

		// Sync commands requiring auth: prompt for confirmation
		if result.RequiresAuth {
			confirmed, err := terminal.Confirm(cmd, result)
			if err == terminal.ErrQuitAll {
				fmt.Println("✗ 取消全部操作")
				return nil
			}
			if err != nil {
				return fmt.Errorf("confirmation error: %w", err)
			}
			if !confirmed {
				continue
			}
		}

		fmt.Printf("\n🔧 Executing [%d/%d]: %s %v\n", i+1, len(intent.Commands), cmd.Cmd, cmd.Args)

		execResult, err := e.executor.Execute(ctx, cmd)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		// Show output (truncated if too long)
		e.displayOutput(execResult.Output)

		// Step 4: Analyze result
		if execResult.Error != nil {
			fmt.Printf("📊 Command failed (exit code %d)\n", execResult.ExitCode)
		} else {
			analysis, err := e.ai.AnalyzeOutput(ctx, cmd.Cmd, execResult.Output)
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
