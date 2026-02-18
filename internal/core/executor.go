package core

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Executor handles command execution
type Executor struct {
	timeout time.Duration
}

// NewExecutor creates a new executor
func NewExecutor(timeout time.Duration) *Executor {
	return &Executor{
		timeout: timeout,
	}
}

// Result represents command execution result
type Result struct {
	Output   string
	ExitCode int
	Error    error
}

// Execute runs a command and returns the result
func (e *Executor) Execute(ctx context.Context, cmd Command) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Build command with args
	cmdParts := append([]string{cmd.Cmd}, cmd.Args...)
	execCmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	err := execCmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}

	result := &Result{
		Output: strings.TrimSpace(output),
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		}
		result.Error = err
	}

	return result, nil
}

// ExecuteBatch runs multiple commands sequentially
func (e *Executor) ExecuteBatch(ctx context.Context, commands []Command) ([]*Result, error) {
	results := make([]*Result, len(commands))

	for i, cmd := range commands {
		result, err := e.Execute(ctx, cmd)
		if err != nil {
			return results, fmt.Errorf("command %d failed: %w", i+1, err)
		}
		results[i] = result

		// Stop if command failed
		if result.Error != nil {
			break
		}
	}

	return results, nil
}
