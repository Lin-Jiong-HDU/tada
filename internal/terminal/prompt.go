package terminal

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

// Errors returned by Confirm
var (
	ErrQuitAll = errors.New("quit all commands")
)

// Confirm prompts the user for command confirmation
// Returns true if approved, false if skipped, ErrQuitAll if quit all
func Confirm(cmd ai.Command, checkResult *security.CheckResult) (bool, error) {
	return ConfirmWithIO(cmd, checkResult, nil, nil)
}

// ConfirmWithIO prompts the user with provided IO (for testing)
func ConfirmWithIO(cmd ai.Command, checkResult *security.CheckResult, input io.Reader, output io.Writer) (bool, error) {
	if input == nil {
		input = os.Stdin
	}
	if output == nil {
		output = os.Stdout
	}

	// Build command string
	cmdStr := cmd.Cmd
	if len(cmd.Args) > 0 {
		cmdStr += " " + strings.Join(cmd.Args, " ")
	}

	// Display prompt
	fmt.Fprintf(output, "\n⚠️  此操作需要您的授权\n\n")
	fmt.Fprintf(output, "命令: %s\n", cmdStr)

	if checkResult.Warning != "" {
		fmt.Fprintf(output, "警告: %s\n", checkResult.Warning)
	}

	if checkResult.Reason != "" {
		fmt.Fprintf(output, "原因: %s\n", checkResult.Reason)
	}

	fmt.Fprintf(output, "\n[y] 执行  [s] 跳过  [q] 取消全部\n> ")

	// Read input
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		choice := strings.ToLower(strings.TrimSpace(scanner.Text()))

		switch choice {
		case "y":
			fmt.Fprintln(output, "✓ 已授权执行")
			return true, nil
		case "s":
			fmt.Fprintln(output, "⊘ 已跳过")
			return false, nil
		case "q":
			fmt.Fprintln(output, "✗ 取消全部操作")
			return false, ErrQuitAll
		default:
			fmt.Fprintf(output, "无效选项，请输入 y/s/q: ")
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}
