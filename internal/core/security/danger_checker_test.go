package security

import (
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

func TestDangerousCommandChecker_IsDangerous(t *testing.T) {
	checker := NewDangerousCommandChecker()

	tests := []struct {
		name      string
		cmd       ai.Command
		dangerous bool
	}{
		{
			name:      "rm -rf / is dangerous",
			cmd:       ai.Command{Cmd: "rm", Args: []string{"-rf", "/"}},
			dangerous: true,
		},
		{
			name:      "ls is not dangerous",
			cmd:       ai.Command{Cmd: "ls", Args: []string{}},
			dangerous: false,
		},
		{
			name:      "chmod is dangerous",
			cmd:       ai.Command{Cmd: "chmod", Args: []string{"777", "file"}},
			dangerous: true,
		},
		{
			name:      "dd is dangerous",
			cmd:       ai.Command{Cmd: "dd", Args: []string{"if=/dev/zero", "of=/dev/sda"}},
			dangerous: true,
		},
		{
			name:      "echo is not dangerous",
			cmd:       ai.Command{Cmd: "echo", Args: []string{"hello"}},
			dangerous: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.IsDangerous(tt.cmd)
			if result != tt.dangerous {
				t.Errorf("IsDangerous() = %v, want %v", result, tt.dangerous)
			}
		})
	}
}

func TestDangerousCommandChecker_IsDangerousPattern(t *testing.T) {
	checker := NewDangerousCommandChecker()

	tests := []struct {
		name      string
		cmd       ai.Command
		dangerous bool
	}{
		{
			name:      "rm -rf .* pattern",
			cmd:       ai.Command{Cmd: "rm", Args: []string{"-rf", ".*"}},
			dangerous: true,
		},
		{
			name:      "redirect to root",
			cmd:       ai.Command{Cmd: "sh", Args: []string{"-c", "echo > /file"}},
			dangerous: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.IsDangerous(tt.cmd)
			if result != tt.dangerous {
				t.Errorf("IsDangerous() = %v, want %v", result, tt.dangerous)
			}
		})
	}
}

func TestDangerousCommandChecker_CommandNormalization(t *testing.T) {
	checker := NewDangerousCommandChecker()

	tests := []struct {
		name      string
		cmd       ai.Command
		dangerous bool
	}{
		{
			name:      "absolute path /bin/rm is dangerous",
			cmd:       ai.Command{Cmd: "/bin/rm", Args: []string{"-rf", "file"}},
			dangerous: true,
		},
		{
			name:      "absolute path /usr/bin/chmod is dangerous",
			cmd:       ai.Command{Cmd: "/usr/bin/chmod", Args: []string{"777", "file"}},
			dangerous: true,
		},
		{
			name:      "rmfoo is NOT dangerous (false positive check)",
			cmd:       ai.Command{Cmd: "rmfoo", Args: []string{}},
			dangerous: false,
		},
		{
			name:      "chmodscript is NOT dangerous",
			cmd:       ai.Command{Cmd: "chmodscript", Args: []string{}},
			dangerous: false,
		},
		{
			name:      "relative path ./rm is dangerous",
			cmd:       ai.Command{Cmd: "./rm", Args: []string{}},
			dangerous: true, // Executing a file named "rm" is suspicious
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.IsDangerous(tt.cmd)
			if result != tt.dangerous {
				t.Errorf("IsDangerous() = %v, want %v", result, tt.dangerous)
			}
		})
	}
}

func TestDangerousCommandChecker_NoSpaceRedirects(t *testing.T) {
	checker := NewDangerousCommandChecker()

	tests := []struct {
		name      string
		cmd       ai.Command
		dangerous bool
	}{
		{
			name:      "chmod 777/ pattern (no space)",
			cmd:       ai.Command{Cmd: "chmod", Args: []string{"777/", "file"}},
			dangerous: true,
		},
		{
			name:      "redirect to root without space",
			cmd:       ai.Command{Cmd: "sh", Args: []string{"-c", "echo >/file"}},
			dangerous: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.IsDangerous(tt.cmd)
			if result != tt.dangerous {
				t.Errorf("IsDangerous() = %v, want %v", result, tt.dangerous)
			}
		})
	}
}
