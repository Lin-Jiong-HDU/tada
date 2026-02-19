package security

import (
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
		dangerousCommands: []string{
			"rm", "rmdir", "dd", "mkfs", "format",
			"chmod", "chown", "userdel", "groupdel",
			"mkfs.", "format.", "fdisk",
		},
		dangerousPatterns: []string{
			"rm -rf /",
			"rm -rf .*",
			"> /",
			">: *",
			"chmod 777 /",
		},
	}
}

// IsDangerous checks if a command is dangerous.
func (dc *DangerousCommandChecker) IsDangerous(cmd ai.Command) bool {
	// Check if command is in dangerous list
	for _, dangerous := range dc.dangerousCommands {
		if strings.HasPrefix(cmd.Cmd, dangerous) {
			return true
		}
	}

	// Build full command string for pattern matching
	cmdStr := cmd.Cmd
	if len(cmd.Args) > 0 {
		cmdStr += " " + strings.Join(cmd.Args, " ")
	}

	// Check dangerous patterns
	for _, pattern := range dc.dangerousPatterns {
		if strings.Contains(cmdStr, pattern) {
			return true
		}
	}

	return false
}
