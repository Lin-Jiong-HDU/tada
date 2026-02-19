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
		// Simple heuristic: commands that typically write
		writeCommands := []string{"rm", "mv", "cp", "touch", "mkdir", "chmod", "chown", "echo", "cat", "tee"}
		isWrite := false
		for _, wc := range writeCommands {
			if cmd.Cmd == wc {
				isWrite = true
				break
			}
		}

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
