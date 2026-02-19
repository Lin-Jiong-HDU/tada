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
	// Canonicalize the checked path (resolve symlinks)
	canonicalPath, err := pc.canonicalizePath(checkPath)
	if err != nil {
		return false
	}

	for _, restricted := range pc.restricted {
		// Canonicalize the restricted path
		canonicalRestricted, err := pc.canonicalizePath(restricted)
		if err != nil {
			continue
		}

		// Check if path is restricted or under restricted directory
		if strings.HasPrefix(canonicalPath, canonicalRestricted+string(filepath.Separator)) ||
			canonicalPath == canonicalRestricted {
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

	// Canonicalize the checked path (resolve symlinks)
	canonicalPath, err := pc.canonicalizePath(checkPath)
	if err != nil {
		return false
	}

	for _, ro := range pc.readonly {
		// Canonicalize the readonly path
		canonicalRO, err := pc.canonicalizePath(ro)
		if err != nil {
			continue
		}

		// Check if path is in readonly directory
		if strings.HasPrefix(canonicalPath, canonicalRO+string(filepath.Separator)) ||
			canonicalPath == canonicalRO {
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

// canonicalizePath expands home directory, converts to absolute path,
// and resolves symlinks to prevent bypass via symlink attacks.
func (pc *PathAccessChecker) canonicalizePath(path string) (string, error) {
	// Expand home directory
	expandedPath := path
	if strings.HasPrefix(expandedPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		expandedPath = filepath.Join(home, path[2:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return "", err
	}

	// Try to resolve symlinks to prevent bypass attacks
	// Walk up the path until we find a directory that exists
	canonicalPath, err := pc.resolveSymlinksWalkUp(absPath)
	if err != nil {
		// Fall back to absolute path if all else fails
		return absPath, nil
	}

	return canonicalPath, nil
}

// resolveSymlinksWalkUp walks up the directory tree resolving symlinks
// until we find a path that exists, then rebuilds the path.
func (pc *PathAccessChecker) resolveSymlinksWalkUp(path string) (string, error) {
	// First, try to resolve the full path directly
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}

	// Path doesn't exist - check parent directory
	parent := filepath.Dir(path)
	base := filepath.Base(path)

	// If we've reached the root, return the path as-is
	if parent == path {
		return path, nil
	}

	// Recursively resolve parent directory
	resolvedParent, err := pc.resolveSymlinksWalkUp(parent)
	if err != nil {
		return "", err
	}

	// Rebuild the path with the resolved parent
	return filepath.Join(resolvedParent, base), nil
}
