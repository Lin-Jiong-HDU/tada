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
	// First, collect all security issues
	var warnings []string
	var reasons []string
	isDangerous := false

	// Check 1: Dangerous command detection
	if sc.dangerChecker.IsDangerous(cmd) {
		isDangerous = true
		warnings = append(warnings, fmt.Sprintf("Dangerous command: %s %v", cmd.Cmd, cmd.Args))
		reasons = append(reasons, "Command is in the dangerous list")
	}

	// Check 2: Path access control
	paths := sc.pathChecker.ExtractPaths(cmd)
	for _, p := range paths {
		// Check restricted paths (always blocks)
		if sc.pathChecker.IsRestricted(p) {
			return &CheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Access denied: %s is restricted", p),
			}, nil
		}

		// Check readonly paths (for write operations)
		isWrite := sc.isWriteOperation(cmd)
		if sc.pathChecker.IsReadOnly(p, isWrite) {
			isDangerous = true
			warnings = append(warnings, fmt.Sprintf("Read-only protection: %s cannot be written", p))
			reasons = append(reasons, "Path is in readonly list")
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
		isDangerous = true
		warnings = append(warnings, shellResult.Warning)
		reasons = append(reasons, shellResult.Reason)
	}

	// Apply CommandLevel policy
	requiresAuth := sc.shouldRequireAuth(isDangerous)

	// Build final result
	result := &CheckResult{
		Allowed:      true,
		RequiresAuth: requiresAuth,
	}

	if requiresAuth && len(warnings) > 0 {
		result.Warning = strings.Join(warnings, "; ")
	}
	if len(reasons) > 0 {
		result.Reason = strings.Join(reasons, "; ")
	}

	return result, nil
}

// CheckPathAccess checks if a path can be accessed.
func (sc *SecurityController) CheckPathAccess(path string, write bool) (*CheckResult, error) {
	// Check restricted (always blocks)
	if sc.pathChecker.IsRestricted(path) {
		return &CheckResult{
			Allowed: false,
			Reason:  fmt.Sprintf("Path %s is restricted", path),
		}, nil
	}

	// Check readonly
	isDangerous := sc.pathChecker.IsReadOnly(path, write)

	// Apply CommandLevel policy
	requiresAuth := sc.shouldRequireAuth(isDangerous)

	result := &CheckResult{
		Allowed:      true,
		RequiresAuth: requiresAuth,
	}

	if requiresAuth && isDangerous {
		result.Warning = fmt.Sprintf("Path %s is read-only", path)
		result.Reason = "Write operation on read-only path"
	}

	return result, nil
}

// AnalyzeShellCommand analyzes a shell command.
func (sc *SecurityController) AnalyzeShellCommand(cmdStr string) (*CheckResult, error) {
	return sc.shellAnalyzer.Analyze(cmdStr), nil
}

// shouldRequireAuth determines if a command requires authorization based on
// the CommandLevel policy and whether the command is dangerous.
func (sc *SecurityController) shouldRequireAuth(isDangerous bool) bool {
	switch sc.policy.CommandLevel {
	case ConfirmAlways:
		// Always require confirmation, regardless of danger
		return true
	case ConfirmNever:
		// Never require confirmation (not recommended for production)
		return false
	case ConfirmDangerous:
		// Only require confirmation for dangerous commands
		return isDangerous
	default:
		// Default to safest behavior: require auth for dangerous commands
		return isDangerous
	}
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
