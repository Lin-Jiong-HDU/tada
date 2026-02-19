// Package security provides security control for command execution.
//
// The security controller sits between the AI parser and command executor,
// protecting users from dangerous operations through:
//
//   - Dangerous command detection (built-in list + AI judgment)
//   - Path access control (restricted + readonly paths)
//   - Shell command analysis (safe/dangerous operations)
//
// Phase 2 implements the security checking logic. TUI-based authorization
// is deferred to Phase 3.
package security
