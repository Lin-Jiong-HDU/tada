package security

import (
	"fmt"
	"strings"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// SecurityController coordinates all security checks.
type SecurityController struct {
	policy        *SecurityPolicy
	dangerChecker *DangerousCommandChecker
	pathChecker   *PathAccessChecker
	shellAnalyzer *ShellCommandAnalyzer
}

// NewSecurityController creates a new security controller.
func NewSecurityController(policy *SecurityPolicy) *SecurityController {
	return &SecurityController{
		policy:        policy,
		dangerChecker: NewDangerousCommandChecker(),
		pathChecker:   NewPathAccessChecker(policy),
		shellAnalyzer: NewShellCommandAnalyzer(policy),
	}
}

// CheckCommand performs comprehensive security check on a command.
func (sc *SecurityController) CheckCommand(cmd ai.Command) (*CheckResult, error) {
	// Check 1: Dangerous command detection
	if sc.dangerChecker.IsDangerous(cmd) {
		return &CheckResult{
			Allowed:      true,
			RequiresAuth: true,
			Warning:      fmt.Sprintf("Dangerous command: %s %v", cmd.Cmd, cmd.Args),
			Reason:       "Command is in the dangerous list",
		}, nil
	}

	// Check 2: Path access control
	paths := sc.pathChecker.ExtractPaths(cmd)
	for _, p := range paths {
		// Check restricted paths
		if sc.pathChecker.IsRestricted(p) {
			return &CheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Access denied: %s is restricted", p),
			}, nil
		}

		// Check readonly paths (for write operations)
		// Determine if this is a write operation by checking for:
		// 1. Redirect operators in command arguments (>, >>)
		// 2. Commands that are inherently write operations
		isWrite := sc.isWriteOperation(cmd)

		if sc.pathChecker.IsReadOnly(p, isWrite) {
			return &CheckResult{
				Allowed:      true,
				RequiresAuth: true,
				Warning:      fmt.Sprintf("Read-only protection: %s cannot be written", p),
				Reason:       "Path is in readonly list",
			}, nil
		}
	}

	// Check 3: Shell command analysis
	cmdStr := cmd.Cmd
	if len(cmd.Args) > 0 {
		cmdStr += " " + strings.Join(cmd.Args, " ")
	}
	shellResult := sc.shellAnalyzer.Analyze(cmdStr)
	if !shellResult.Allowed {
		return shellResult, nil
	}
	if shellResult.RequiresAuth {
		return shellResult, nil
	}

	// All checks passed
	return &CheckResult{
		Allowed: true,
	}, nil
}

// CheckPathAccess checks if a path can be accessed.
func (sc *SecurityController) CheckPathAccess(path string, write bool) (*CheckResult, error) {
	// Check restricted
	if sc.pathChecker.IsRestricted(path) {
		return &CheckResult{
			Allowed: false,
			Reason:  fmt.Sprintf("Path %s is restricted", path),
		}, nil
	}

	// Check readonly
	if sc.pathChecker.IsReadOnly(path, write) {
		return &CheckResult{
			Allowed:      true,
			RequiresAuth: true,
			Warning:      fmt.Sprintf("Path %s is read-only", path),
			Reason:       "Write operation on read-only path",
		}, nil
	}

	return &CheckResult{Allowed: true}, nil
}

// AnalyzeShellCommand analyzes a shell command.
func (sc *SecurityController) AnalyzeShellCommand(cmdStr string) (*CheckResult, error) {
	return sc.shellAnalyzer.Analyze(cmdStr), nil
}

// isWriteOperation determines if a command is a write operation.
// It checks for redirect operators and inherently write-focused commands.
func (sc *SecurityController) isWriteOperation(cmd ai.Command) bool {
	// Build full command string to check for redirects
	cmdStr := cmd.Cmd
	if len(cmd.Args) > 0 {
		cmdStr += " " + strings.Join(cmd.Args, " ")
	}

	// Check for redirect operators (output redirection)
	if strings.Contains(cmdStr, ">") || strings.Contains(cmdStr, ">>") {
		return true
	}

	// Commands that are inherently write operations
	// (removed echo and cat since they're not inherently writes)
	writeCommands := map[string]bool{
		"rm":    true, // delete
		"mv":    true, // move/rename
		"cp":    true, // copy
		"touch": true, // create file
		"mkdir": true, // create directory
		"chmod": true, // change permissions
		"chown": true, // change owner
		"tee":   true, // write to stdin and file
	}

	return writeCommands[cmd.Cmd]
}
