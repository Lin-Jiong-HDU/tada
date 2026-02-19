package security

// SecurityPolicy defines the security configuration.
type SecurityPolicy struct {
	// CommandLevel determines when commands require confirmation.
	// "always" - every command requires confirmation
	// "dangerous" - only dangerous commands require confirmation
	// "never" - no confirmation required
	CommandLevel ConfirmLevel `mapstructure:"command_level"`

	// RestrictedPaths contains paths that are completely forbidden.
	RestrictedPaths []string `mapstructure:"restricted_paths"`

	// ReadOnlyPaths contains paths that cannot be written to.
	ReadOnlyPaths []string `mapstructure:"readonly_paths"`

	// AllowShell determines if shell commands (pipes, redirects) are allowed.
	AllowShell bool `mapstructure:"allow_shell"`

	// AllowTerminalTakeover determines if multi-step operations are allowed.
	AllowTerminalTakeover bool `mapstructure:"allow_terminal_takeover"`
}

// ConfirmLevel represents the command confirmation level.
type ConfirmLevel string

const (
	ConfirmAlways    ConfirmLevel = "always"
	ConfirmDangerous ConfirmLevel = "dangerous"
	ConfirmNever     ConfirmLevel = "never"
)

// DefaultPolicy returns the default security policy (balanced mode).
func DefaultPolicy() *SecurityPolicy {
	return &SecurityPolicy{
		CommandLevel:          ConfirmDangerous,
		RestrictedPaths:       []string{},
		ReadOnlyPaths:         []string{},
		AllowShell:            true,
		AllowTerminalTakeover: true,
	}
}
