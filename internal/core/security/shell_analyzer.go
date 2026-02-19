package security

import (
	"regexp"
	"strings"
)

// ShellCommandAnalyzer analyzes shell commands for safety.
type ShellCommandAnalyzer struct {
	allowShell bool
}

// NewShellCommandAnalyzer creates a new shell analyzer.
func NewShellCommandAnalyzer(policy *SecurityPolicy) *ShellCommandAnalyzer {
	return &ShellCommandAnalyzer{
		allowShell: policy.AllowShell,
	}
}

// CheckResult represents the result of a security check.
type CheckResult struct {
	Allowed      bool
	RequiresAuth bool
	Warning      string
	Reason       string
}

// Analyze analyzes a shell command string for safety.
func (sa *ShellCommandAnalyzer) Analyze(cmdStr string) *CheckResult {
	// Check if shell is allowed
	if !sa.allowShell {
		return &CheckResult{
			Allowed: false,
			Reason:  "Shell commands are disabled (allow_shell=false)",
		}
	}

	// Check for path traversal patterns
	if strings.Contains(cmdStr, "../") {
		return &CheckResult{
			Allowed:      true,
			RequiresAuth: true,
			Warning:      "Dangerous shell operation detected",
			Reason:       "potential path traversal",
		}
	}

	// Check for dangerous redirects to protected paths
	if sa.hasDangerousRedirect(cmdStr) {
		return &CheckResult{
			Allowed:      true,
			RequiresAuth: true,
			Warning:      "Dangerous shell operation detected",
			Reason:       "redirecting to protected system path",
		}
	}

	// Safe shell operation
	return &CheckResult{
		Allowed: true,
	}
}

// hasDangerousRedirect checks if the command has redirects to protected paths.
// It detects redirects regardless of whitespace or file descriptor prefixes.
func (sa *ShellCommandAnalyzer) hasDangerousRedirect(cmdStr string) bool {
	// Regex to match shell redirects with their target paths
	// Matches: >, >>, <, with optional file descriptor (0-9) and optional whitespace
	// Examples: >file, > file, 1>/etc/passwd, 2>> /var/log, >>file
	redirectRegex := regexp.MustCompile(`[0-9]?(>>?>?)[ \t]*([^\s&|;]+)`)

	matches := redirectRegex.FindAllStringSubmatch(cmdStr, -1)

	// Protected paths that require authorization
	protectedPaths := []string{
		"/etc/",
		"/usr/",
		"/usr/bin/",
		"/usr/sbin/",
		"/System",
		"/bin/",
		"/sbin/",
		"/boot/",
		"/lib/",
		"/lib64/",
	}

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		redirectOp := match[1]
		targetPath := match[2]

		// Only check output redirects (>, >>)
		if redirectOp == ">" || redirectOp == ">>" {
			for _, protected := range protectedPaths {
				// Check if target is a protected path or under a protected directory
				if strings.HasPrefix(targetPath, protected) || targetPath == strings.TrimSuffix(protected, "/") {
					return true
				}
			}
		}
	}

	return false
}
