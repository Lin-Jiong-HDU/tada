package security

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// PathAccessChecker checks path access permissions.
type PathAccessChecker struct {
	restricted []string
	readonly   []string
}

// NewPathAccessChecker creates a new path checker.
func NewPathAccessChecker(policy *SecurityPolicy) *PathAccessChecker {
	return &PathAccessChecker{
		restricted: policy.RestrictedPaths,
		readonly:   policy.ReadOnlyPaths,
	}
}

// IsRestricted checks if a path is restricted.
func (pc *PathAccessChecker) IsRestricted(checkPath string) bool {
	// Expand home directory
	expandedPath := checkPath
	if strings.HasPrefix(expandedPath, "~/") {
		home, _ := os.UserHomeDir()
		expandedPath = filepath.Join(home, checkPath[2:])
	}

	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return false
	}

	for _, restricted := range pc.restricted {
		expandedRestricted := restricted
		if strings.HasPrefix(restricted, "~/") {
			home, _ := os.UserHomeDir()
			expandedRestricted = filepath.Join(home, restricted[2:])
		}

		absRestricted, err := filepath.Abs(expandedRestricted)
		if err != nil {
			continue
		}

		// Check if path is restricted or under restricted directory
		if strings.HasPrefix(absPath, absRestricted+string(filepath.Separator)) ||
			absPath == absRestricted {
			return true
		}
	}

	return false
}

// IsReadOnly checks if a path is readonly (for write operations).
func (pc *PathAccessChecker) IsReadOnly(checkPath string, write bool) bool {
	if !write {
		return false
	}

	// Expand home directory
	expandedPath := checkPath
	if strings.HasPrefix(expandedPath, "~/") {
		home, _ := os.UserHomeDir()
		expandedPath = filepath.Join(home, checkPath[2:])
	}

	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return false
	}

	for _, ro := range pc.readonly {
		expandedRO := ro
		if strings.HasPrefix(ro, "~/") {
			home, _ := os.UserHomeDir()
			expandedRO = filepath.Join(home, ro[2:])
		}

		absRO, err := filepath.Abs(expandedRO)
		if err != nil {
			continue
		}

		// Check if path is in readonly directory
		if strings.HasPrefix(absPath, absRO+string(filepath.Separator)) ||
			absPath == absRO {
			return true
		}
	}

	return false
}

// ExtractPaths extracts file paths from command arguments.
func (pc *PathAccessChecker) ExtractPaths(cmd ai.Command) []string {
	var paths []string

	allArgs := append([]string{cmd.Cmd}, cmd.Args...)

	for _, arg := range allArgs {
		// Skip flags and options
		if strings.HasPrefix(arg, "-") {
			continue
		}

		// Check if it looks like a path
		if strings.Contains(arg, "/") || strings.Contains(arg, "~") {
			paths = append(paths, arg)
		}
	}

	return paths
}
