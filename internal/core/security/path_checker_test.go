package security

import (
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

func TestPathAccessChecker_IsRestricted(t *testing.T) {
	policy := &SecurityPolicy{
		RestrictedPaths: []string{"/etc", "/usr/bin"},
	}
	checker := NewPathAccessChecker(policy)

	tests := []struct {
		name       string
		path       string
		restricted bool
	}{
		{"etc is restricted", "/etc/passwd", true},
		{"usr/bin is restricted", "/usr/bin/ls", true},
		{"home is not restricted", "/home/user/file.txt", false},
		{"subdir of restricted", "/etc/config/file", true},
		{"exact match restricted path", "/etc/passwd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.IsRestricted(tt.path)
			if result != tt.restricted {
				t.Errorf("IsRestricted(%s) = %v, want %v", tt.path, result, tt.restricted)
			}
		})
	}
}

func TestPathAccessChecker_IsReadOnly(t *testing.T) {
	policy := &SecurityPolicy{
		ReadOnlyPaths: []string{"~/.ssh", "/.gnupg"},
	}
	checker := NewPathAccessChecker(policy)

	tests := []struct {
		name     string
		path     string
		write    bool
		readonly bool
	}{
		{"read ssh key", "~/.ssh/id_rsa", false, false},
		{"write ssh key", "~/.ssh/id_rsa.pub", true, true},
		{"write to normal dir", "/tmp/file", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.IsReadOnly(tt.path, tt.write)
			if result != tt.readonly {
				t.Errorf("IsReadOnly(%s, write=%v) = %v, want %v", tt.path, tt.write, result, tt.readonly)
			}
		})
	}
}

func TestPathAccessChecker_ExtractPaths(t *testing.T) {
	checker := NewPathAccessChecker(&SecurityPolicy{})

	tests := []struct {
		name  string
		cmd   ai.Command
		paths []string
	}{
		{
			name:  "single file",
			cmd:   ai.Command{Cmd: "cat", Args: []string{"/etc/passwd"}},
			paths: []string{"/etc/passwd"},
		},
		{
			name:  "multiple files",
			cmd:   ai.Command{Cmd: "ls", Args: []string{"/etc", "/home"}},
			paths: []string{"/etc", "/home"},
		},
		{
			name:  "no paths",
			cmd:   ai.Command{Cmd: "echo", Args: []string{"hello"}},
			paths: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.ExtractPaths(tt.cmd)
			if len(result) != len(tt.paths) {
				t.Errorf("ExtractPaths() returned %d paths, want %d", len(result), len(tt.paths))
			}
		})
	}
}
