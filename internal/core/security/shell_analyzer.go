package security

import (
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

	// Check for dangerous patterns
	dangerousPatterns := []struct {
		pattern string
		reason  string
	}{
		{"> /etc/", "redirecting to system path /etc/"},
		{"> /usr/", "redirecting to system path /usr/"},
		{"> /System", "redirecting to System directory"},
		{"../", "potential path traversal"},
	}

	for _, dp := range dangerousPatterns {
		if strings.Contains(cmdStr, dp.pattern) {
			return &CheckResult{
				Allowed:      true,
				RequiresAuth: true,
				Warning:      "Dangerous shell operation detected",
				Reason:       dp.reason,
			}
		}
	}

	// Safe shell operation
	return &CheckResult{
		Allowed: true,
	}
}
