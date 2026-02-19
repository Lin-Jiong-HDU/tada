package security

import (
	"testing"
)

func TestShellCommandAnalyzer_Analyze(t *testing.T) {
	policy := &SecurityPolicy{AllowShell: true}
	analyzer := NewShellCommandAnalyzer(policy)

	tests := []struct {
		name          string
		cmdStr        string
		requiresAuth bool
		reason        string
	}{
		{
			name:          "safe pipe",
			cmdStr:        "ls | grep test",
			requiresAuth: false,
			reason:        "",
		},
		{
			name:          "safe redirect",
			cmdStr:        "echo hello > /tmp/file",
			requiresAuth: false,
			reason:        "",
		},
		{
			name:          "dangerous system redirect",
			cmdStr:        "cat file > /etc/config",
			requiresAuth: true,
			reason:        "redirecting to system path /etc/",
		},
		{
			name:          "path traversal",
			cmdStr:        "cat file > ../../../../etc/passwd",
			requiresAuth: true,
			reason:        "potential path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.Analyze(tt.cmdStr)
			if result.RequiresAuth != tt.requiresAuth {
				t.Errorf("Analyze(%s).RequiresAuth = %v, want %v", tt.cmdStr, result.RequiresAuth, tt.requiresAuth)
			}
		})
	}
}

func TestShellCommandAnalyzer_ShellDisabled(t *testing.T) {
	policy := &SecurityPolicy{AllowShell: false}
	analyzer := NewShellCommandAnalyzer(policy)

	result := analyzer.Analyze("ls | grep test")
	if result.Allowed {
		t.Error("Expected shell commands to be rejected when AllowShell=false")
	}
}
