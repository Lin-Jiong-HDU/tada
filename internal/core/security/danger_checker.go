package security

import (
	"path/filepath"
	"strings"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// DangerousCommandChecker detects dangerous commands.
type DangerousCommandChecker struct {
	dangerousCommands []string
	dangerousPatterns []string
}

// NewDangerousCommandChecker creates a new danger checker.
func NewDangerousCommandChecker() *DangerousCommandChecker {
	return &DangerousCommandChecker{
		// Use exact command names - matching is done with filepath.Base()
		dangerousCommands: []string{
			"rm", "rmdir", "dd",
			"mkfs", "format",
			"chmod", "chown",
			"userdel", "groupdel",
			"fdisk",
		},
		// Patterns use exact string matching - these are specific dangerous
		// command combinations that should always require authorization
		dangerousPatterns: []string{
			"rm -rf /",
			"rm -rf .*",
			"> /",
			">/",           // No-space variant
			"chmod 777 /",
			"chmod 777/",   // No-space variant
		},
	}
}

// IsDangerous checks if a command is dangerous.
func (dc *DangerousCommandChecker) IsDangerous(cmd ai.Command) bool {
	// Normalize the command name by extracting the basename
	// This handles both "rm" and "/bin/rm" correctly
	cmdName := filepath.Base(cmd.Cmd)

	// Check if command is in dangerous list (exact match)
	for _, dangerous := range dc.dangerousCommands {
		if cmdName == dangerous {
			return true
		}
	}

	// Build full command string for pattern matching
	cmdStr := cmd.Cmd
	if len(cmd.Args) > 0 {
		cmdStr += " " + strings.Join(cmd.Args, " ")
	}

	// Check dangerous patterns (substring match)
	for _, pattern := range dc.dangerousPatterns {
		if strings.Contains(cmdStr, pattern) {
			return true
		}
	}

	return false
}
