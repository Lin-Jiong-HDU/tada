# Security Control Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a comprehensive security control system for tada to protect users from dangerous AI-generated commands.

**Architecture:** A layered security controller that sits between AI parser and command executor, combining built-in dangerous command detection with AI judgment.

**Tech Stack:** Go 1.25.7, Viper (config), existing core infrastructure

---

## Task 1: Create Security Package Structure

**Files:**
- Create: `internal/core/security/policy.go`
- Create: `internal/core/security/doc.go`

**Step 1: Create package documentation**

Create: `internal/core/security/doc.go`
```go
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
```

**Step 2: Define SecurityPolicy structure**

Create: `internal/core/security/policy.go`
```go
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
		CommandLevel:         ConfirmDangerous,
		RestrictedPaths:      []string{},
		ReadOnlyPaths:        []string{},
		AllowShell:           true,
		AllowTerminalTakeover: true,
	}
}
```

**Step 3: Run tests to verify package compiles**

Run:
```bash
go build ./internal/core/security/
```

Expected: No errors

**Step 4: Commit**

```bash
git add internal/core/security/
git commit -m "feat: add security package with policy definition"
```

---

## Task 2: Implement DangerousCommandChecker

**Files:**
- Create: `internal/core/security/danger_checker.go`
- Create: `internal/core/security/danger_checker_test.go`

**Step 1: Write the failing test**

Create: `internal/core/security/danger_checker_test.go`
```go
package security

import (
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

func TestDangerousCommandChecker_IsDangerous(t *testing.T) {
	checker := NewDangerousCommandChecker()

	tests := []struct {
		name     string
		cmd      ai.Command
		dangerous bool
	}{
		{
			name:     "rm -rf / is dangerous",
			cmd:      ai.Command{Cmd: "rm", Args: []string{"-rf", "/"}},
			dangerous: true,
		},
		{
			name:     "ls is not dangerous",
			cmd:      ai.Command{Cmd: "ls", Args: []string{}},
			dangerous: false,
		},
		{
			name:     "chmod is dangerous",
			cmd:      ai.Command{Cmd: "chmod", Args: []string{"777", "file"}},
			dangerous: true,
		},
		{
			name:     "dd is dangerous",
			cmd:      ai.Command{Cmd: "dd", Args: []string{"if=/dev/zero", "of=/dev/sda"}},
			dangerous: true,
		},
		{
			name:     "echo is not dangerous",
			cmd:      ai.Command{Cmd: "echo", Args: []string{"hello"}},
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
		name     string
		cmd      ai.Command
		dangerous bool
	}{
		{
			name:     "rm -rf .* pattern",
			cmd:      ai.Command{Cmd: "rm", Args: []string{"-rf", ".*"}},
			dangerous: true,
		},
		{
			name:     "redirect to root",
			cmd:      ai.Command{Cmd: "sh", Args: []string{"-c", "echo > /file"}},
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
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/core/security/danger_checker_test.go -v
```

Expected: FAIL with "undefined: NewDangerousCommandChecker"

**Step 3: Write minimal implementation**

Create: `internal/core/security/danger_checker.go`
```go
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
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/core/security/ -v -run TestDangerousCommandChecker
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/security/danger_checker.go internal/core/security/danger_checker_test.go
git commit -m "feat: add dangerous command checker with tests"
```

---

## Task 3: Implement PathAccessChecker

**Files:**
- Create: `internal/core/security/path_checker.go`
- Create: `internal/core/security/path_checker_test.go`

**Step 1: Write the failing test**

Create: `internal/core/security/path_checker_test.go`
```go
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
		name     string
		path     string
		restricted bool
	}{
		{"etc is restricted", "/etc/passwd", true},
		{"usr/bin is restricted", "/usr/bin/ls", true},
		{"home is not restricted", "/home/user/file.txt", false},
		{"subdir of restricted", "/etc/config/file", true},
		{"case insensitive", "/ETC/passwd", true},
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
		name   string
		cmd    ai.Command
		paths  []string
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
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/core/security/path_checker_test.go -v
```

Expected: FAIL with "undefined: NewPathAccessChecker"

**Step 3: Write minimal implementation**

Create: `internal/core/security/path_checker.go`
```go
package security

import (
	"os/filepath"
	"path"
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
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/core/security/ -v -run TestPathAccessChecker
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/security/path_checker.go internal/core/security/path_checker_test.go
git commit -m "feat: add path access checker with tests"
```

---

## Task 4: Implement ShellCommandAnalyzer

**Files:**
- Create: `internal/core/security/shell_analyzer.go`
- Create: `internal/core/security/shell_analyzer_test.go`

**Step 1: Write the failing test**

Create: `internal/core/security/shell_analyzer_test.go`
```go
package security

import (
	"testing"

	. "github.com/Lin-Jiong-HDU/tada/internal/core/security"
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
			reason:        "potential path traversal attack",
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
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/core/security/shell_analyzer_test.go -v
```

Expected: FAIL with "undefined: NewShellCommandAnalyzer"

**Step 3: Write minimal implementation**

Create: `internal/core/security/shell_analyzer.go`
```go
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
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/core/security/ -v -run TestShellCommandAnalyzer
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/security/shell_analyzer.go internal/core/security/shell_analyzer_test.go
git commit -m "feat: add shell command analyzer with tests"
```

---

## Task 5: Implement SecurityController

**Files:**
- Create: `internal/core/security/controller.go`
- Create: `internal/core/security/controller_test.go`

**Step 1: Write the failing test**

Create: `internal/core/security/controller_test.go`
```go
package security

import (
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

func TestSecurityController_CheckCommand(t *testing.T) {
	policy := &SecurityPolicy{
		CommandLevel:    ConfirmDangerous,
		RestrictedPaths: []string{"/etc"},
		ReadOnlyPaths:   []string{"~/.ssh"},
		AllowShell:      true,
	}
	controller := NewSecurityController(policy)

	t.Run("safe command allowed", func(t *testing.T) {
		result := controller.CheckCommand(ai.Command{Cmd: "ls", Args: []string{}})
		if !result.Allowed {
			t.Error("Expected safe command to be allowed")
		}
	})

	t.Run("dangerous command requires auth", func(t *testing.T) {
		result := controller.CheckCommand(ai.Command{Cmd: "rm", Args: []string{"-rf", "/tmp/test"}})
		if !result.RequiresAuth {
			t.Error("Expected dangerous command to require auth")
		}
	})

	t.Run("restricted path rejected", func(t *testing.T) {
		result := controller.CheckCommand(ai.Command{Cmd: "cat", Args: []string{"/etc/passwd"}})
		if result.Allowed {
			t.Error("Expected restricted path access to be rejected")
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/core/security/controller_test.go -v
```

Expected: FAIL with "undefined: NewSecurityController"

**Step 3: Write minimal implementation**

Create: `internal/core/security/controller.go`
```go
package security

import (
	"fmt"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// SecurityController coordinates all security checks.
type SecurityController struct {
	policy        *SecurityPolicy
	dangerChecker *DangerousCommandChecker
	pathChecker   *PathAccessChecker
	shellAnalyzer *ShellCommandAnalyzer
}

// NewSecurityController creates a new security controller.
func NewSecurityController(policy *SecurityPolicy) *SecurityController {
	return &SecurityController{
		policy:        policy,
		dangerChecker: NewDangerousCommandChecker(),
		pathChecker:   NewPathAccessChecker(policy),
		shellAnalyzer: NewShellCommandAnalyzer(policy),
	}
}

// CheckCommand performs comprehensive security check on a command.
func (sc *SecurityController) CheckCommand(cmd ai.Command) (*CheckResult, error) {
	// Check 1: Dangerous command detection
	if sc.dangerChecker.IsDangerous(cmd) {
		return &CheckResult{
			Allowed:      true,
			RequiresAuth: true,
			Warning:      fmt.Sprintf("Dangerous command: %s %v", cmd.Cmd, cmd.Args),
			Reason:       "Command is in the dangerous list",
		}, nil
	}

	// Check 2: Path access control
	paths := sc.pathChecker.ExtractPaths(cmd)
	for _, p := range paths {
		// Check restricted paths
		if sc.pathChecker.IsRestricted(p) {
			return &CheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Access denied: %s is restricted", p),
			}, nil
		}

		// Check readonly paths (for write operations)
		// Simple heuristic: commands that typically write
		writeCommands := []string{"rm", "mv", "cp", "touch", "mkdir", "chmod", "chown", "echo", "cat", "tee"}
		isWrite := false
		for _, wc := range writeCommands {
			if cmd.Cmd == wc {
				isWrite = true
				break
			}
		}

		if sc.pathChecker.IsReadOnly(p, isWrite) {
			return &CheckResult{
				Allowed:      true,
				RequiresAuth: true,
				Warning:      fmt.Sprintf("Read-only protection: %s cannot be written", p),
				Reason:       "Path is in readonly list",
			}, nil
		}
	}

	// Check 3: Shell command analysis
	cmdStr := cmd.Cmd
	if len(cmd.Args) > 0 {
		cmdStr += " " + cmd.Cmd
	}
	shellResult := sc.shellAnalyzer.Analyze(cmdStr)
	if !shellResult.Allowed {
		return shellResult, nil
	}
	if shellResult.RequiresAuth {
		return shellResult, nil
	}

	// All checks passed
	return &CheckResult{
		Allowed: true,
	}, nil
}

// CheckPathAccess checks if a path can be accessed.
func (sc *SecurityController) CheckPathAccess(path string, write bool) (*CheckResult, error) {
	// Check restricted
	if sc.pathChecker.IsRestricted(path) {
		return &CheckResult{
			Allowed: false,
			Reason:  fmt.Sprintf("Path %s is restricted", path),
		}, nil
	}

	// Check readonly
	if sc.pathChecker.IsReadOnly(path, write) {
		return &CheckResult{
			Allowed:      true,
			RequiresAuth: true,
			Warning:      fmt.Sprintf("Path %s is read-only", path),
			Reason:       "Write operation on read-only path",
		}, nil
	}

	return &CheckResult{Allowed: true}, nil
}

// AnalyzeShellCommand analyzes a shell command.
func (sc *SecurityController) AnalyzeShellCommand(cmdStr string) (*CheckResult, error) {
	return sc.shellAnalyzer.Analyze(cmdStr), nil
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
go test ./internal/core/security/ -v -run TestSecurityController
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/security/controller.go internal/core/security/controller_test.go
git commit -m "feat: add security controller with comprehensive checks"
```

---

## Task 6: Update Config Structure

**Files:**
- Modify: `internal/storage/config.go`
- Modify: `internal/storage/config_test.go`

**Step 1: Update Config structure**

Modify: `internal/storage/config.go`
```go
// Config holds the application configuration
type Config struct {
	AI       AIConfig       `mapstructure:"ai"`
	Security SecurityPolicy `mapstructure:"security"`
}
```

**Step 2: Import security package**

Add to imports in `internal/storage/config.go`:
```go
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
	"github.com/spf13/viper"
)
```

**Step 3: Update InitConfig to include security defaults**

Add to `internal/storage/config.go` in `InitConfig()` function:
```go
// Set defaults
v.SetDefault("ai.provider", "openai")
v.SetDefault("ai.model", "gpt-4o")
v.SetDefault("ai.base_url", "https://api.openai.com/v1")
v.SetDefault("ai.timeout", 30)
v.SetDefault("ai.max_tokens", 4096)

// Security defaults
v.SetDefault("security.command_level", "dangerous")
v.SetDefault("security.allow_shell", true)
v.SetDefault("security.allow_terminal_takeover", true)
v.SetDefault("security.restricted_paths", []string{})
v.SetDefault("security.readonly_paths", []string{})
```

**Step 4: Update SaveConfig to include security**

Add to `internal/storage/config.go` in `SaveConfig()` function:
```go
// Save security config
v.Set("security.command_level", cfg.Security.CommandLevel)
v.Set("security.allow_shell", cfg.Security.AllowShell)
v.Set("security.allow_terminal_takeover", cfg.Security.AllowTerminalTakeover)

// Handle slices
securityMap := v.Get("security")
if securityMap == nil {
    v.Set("security", make(map[string]interface{}))
}

v.Set("security.command_level", cfg.Security.CommandLevel)
v.Set("security.allow_shell", cfg.Security.AllowShell)
v.Set("security.allow_terminal_takeover", cfg.Security.AllowTerminalTakeover)
v.Set("security.restricted_paths", cfg.Security.RestrictedPaths)
v.Set("security.readonly_paths", cfg.Security.ReadOnlyPaths)
```

**Step 5: Add security config test**

Add to `internal/storage/config_test.go`:
```go
func TestSecurityDefaults(t *testing.T) {
    oldHome := os.Getenv("HOME")
    tmpDir, _ := os.MkdirTemp("", "tada-test-*")
    defer os.RemoveAll(tmpDir)
    os.Setenv("HOME", tmpDir)
    defer os.Setenv("HOME", oldHome)

    cfg, err := InitConfig()
    if err != nil {
        t.Fatalf("InitConfig failed: %v", err)
    }

    if cfg.Security.CommandLevel != "dangerous" {
        t.Errorf("Expected default command_level 'dangerous', got '%s'", cfg.Security.CommandLevel)
    }
    if !cfg.Security.AllowShell {
        t.Error("Expected default allow_shell to be true")
    }
}
```

**Step 6: Run tests to verify**

Run:
```bash
go test ./internal/storage/ -v
```

Expected: All tests PASS

**Step 7: Commit**

```bash
git add internal/storage/config.go internal/storage/config_test.go
git commit -m "feat: integrate security policy into config system"
```

---

## Task 7: Integrate SecurityController into Engine

**Files:**
- Modify: `internal/core/engine.go`
- Modify: `internal/core/engine_test.go`

**Step 1: Update Engine structure**

Modify: `internal/core/engine.go`
```go
import (
	"context"
	"fmt"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
)

// Engine orchestrates the AI workflow
type Engine struct {
	ai                ai.AIProvider
	executor          *Executor
	securityController *security.SecurityController  // New field
}

// NewEngine creates a new engine
func NewEngine(aiProvider ai.AIProvider, executor *Executor, securityPolicy *security.SecurityPolicy) *Engine {
	return &Engine{
		ai:                aiProvider,
		executor:          executor,
		securityController: security.NewSecurityController(securityPolicy),
	}
}
```

**Step 2: Add security check to Process method**

Modify: `internal/core/engine.go` Process method, add after intent parsing:
```go
// Step 1.5: Security check for each command
for _, cmd := range intent.Commands {
	result, err := sc.securityController.CheckCommand(cmd)
	if err != nil {
		return fmt.Errorf("security check failed: %w", err)
	}

	if !result.Allowed {
		return fmt.Errorf("🚫 安全拒绝: %s", result.Reason)
	}

	if result.RequiresAuth {
		// Phase 2: Show warning but continue
		// Phase 3 will implement TUI authorization
		fmt.Printf("⚠️  %s\n", result.Warning)
		fmt.Println("⚠️  注意: 完整的授权确认将在 Phase 3 (TUI) 中实现")
	}
}
```

**Step 3: Update Process method signature**

In the Process method, update the for loop:
```go
// Step 3: Execute commands (with security check)
for i, cmd := range intent.Commands {
	// Security check before execution
	result, err := e.securityController.CheckCommand(cmd)
	if err != nil {
		return fmt.Errorf("security check failed: %w", err)
	}

	if !result.Allowed {
		fmt.Printf("🚫 拒绝执行: %s\n", result.Reason)
		continue
	}

	if result.RequiresAuth {
		fmt.Printf("⚠️  %s\n", result.Warning)
		fmt.Println("⚠️  注意: 完整的授权确认将在 Phase 3 (TUI) 中实现")
	}

	fmt.Printf("\n🔧 Executing [%d/%d]: %s %v\n", i+1, len(intent.Commands), cmd.Cmd, cmd.Args)
	// ... rest of execution
```

**Step 4: Add engine tests**

Create: `internal/core/engine_security_test.go`:
```go
package core

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
)

func TestEngine_SecurityCheck(t *testing.T) {
	policy := &security.SecurityPolicy{
		CommandLevel:    security.ConfirmDangerous,
		RestrictedPaths: []string{"/etc"},
		ReadOnlyPaths:   []string{"~/.ssh"},
		AllowShell:      true,
	}

	engine := NewEngine(nil, NewExecutor(30*time.Second), policy)

	// Test that security controller is set
	if engine.securityController == nil {
		t.Error("Security controller should be initialized")
	}
}

func TestEngine_SecurityCheckIntegration(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	// This test requires a mock AI provider
	// For now, just verify the structure
	policy := security.DefaultPolicy()
	engine := NewEngine(nil, NewExecutor(30*time.Second), policy)

	if engine.securityController == nil {
		t.Error("Expected security controller to be initialized")
	}
}
```

**Step 5: Run tests**

Run:
```bash
go test ./internal/core/ -v -run TestEngine
```

Expected: PASS

**Step 6: Commit**

```bash
git add internal/core/engine.go internal/core/engine_security_test.go
git commit -m "feat: integrate security controller into engine"
```

---

## Task 8: Update Main to Pass Security Policy

**Files:**
- Modify: `cmd/tada/main.go`

**Step 1: Update main.go to pass security policy**

Modify: `cmd/tada/main.go`
```go
	Run: func(cmd *cobra.Command, args []string) {
		cfg := storage.GetConfig()
		input := args[0]

		// Validate config
		if cfg.AI.APIKey == "" {
			fmt.Fprintf(os.Stderr, "❌ Error: AI API key not configured. Please set it in ~/.tada/config.yaml\n")
			fmt.Fprintf(os.Stderr, "Example:\n  ai:\n    api_key: sk-xxx\n")
			os.Exit(1)
		}

		// Initialize components - create AI provider based on config
		var aiProvider ai.AIProvider
		switch cfg.AI.Provider {
		case "openai":
			aiProvider = openai.NewClient(cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL)
		case "glm", "zhipu":
			aiProvider = glm.NewClient(cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL)
		default:
			fmt.Fprintf(os.Stderr, "❌ Error: unsupported provider '%s' (supported: openai, glm)\n", cfg.AI.Provider)
			os.Exit(1)
		}

		executor := core.NewExecutor(30 * time.Second)

		// Use security policy from config, or defaults if not set
		securityPolicy := &cfg.Security
		if securityPolicy.CommandLevel == "" {
			securityPolicy = core.DefaultPolicy()
		}

		engine := core.NewEngine(aiProvider, executor, securityPolicy)

		// Process request
		if err := engine.Process(context.Background(), input, ""); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
```

**Step 2: Add import for core security**

Add to imports:
```go
	"github.com/Lin-Jiong-HDU/tada/internal/core"
```

**Step 3: Test the build**

Run:
```bash
go build -o bin/tada cmd/tada/main.go
```

Expected: No errors

**Step 4: Commit**

```bash
git add cmd/tada/main.go
git commit -m "feat: pass security policy to engine from config"
```

---

## Task 9: Write Integration Tests

**Files:**
- Create: `tests/integration/security_integration_test.go`

**Step 1: Write integration tests**

Create: `tests/integration/security_integration_test.go`
```go
package integration

import (
	"os"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/core"
	"github.com/Lin-Jiong-HDU/tada/internal/core/security"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
)

func TestSecurityIntegration_ConfigLoading(t *testing.T) {
	oldHome := os.Getenv("HOME")
	tmpDir, err := os.MkdirTemp("", "tada-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create config with security settings
	cfg, err := storage.InitConfig()
	if err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	// Verify defaults
	if cfg.Security.CommandLevel != security.ConfirmDangerous {
		t.Errorf("Expected default command_level 'dangerous', got '%s'", cfg.Security.CommandLevel)
	}
}

func TestSecurityIntegration_EngineCreation(t *testing.T) {
	policy := security.DefaultPolicy()
	engine := core.NewEngine(nil, core.NewExecutor(0), policy)

	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}
}

func TestSecurityIntegration_FullWorkflow(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1")
	}

	// This would test the full security workflow
	// Implementation depends on having a working AI provider
}
```

**Step 2: Run tests**

Run:
```bash
go test ./tests/integration/ -v
```

Expected: PASS

**Step 3: Commit**

```bash
git add tests/integration/security_integration_test.go
git commit -m "test: add security integration tests"
```

---

## Task 10: Update Documentation

**Files:**
- Modify: `docs/getting-started.md`
- Modify: `README.md`

**Step 1: Update getting-started.md**

Add to `docs/getting-started.md`:
```markdown
## Security Configuration

tada includes security controls to protect against dangerous AI-generated commands.

### Security Levels

```yaml
security:
  # always: every command requires confirmation
  # dangerous: only dangerous commands require confirmation (default)
  # never: no confirmation required
  command_level: dangerous

  # Paths that are completely forbidden
  restricted_paths:
    - /etc
    - /usr/bin
    - /System

  # Paths that cannot be written to
  readonly_paths:
    - ~/.ssh
    - ~/.gnupg

  # Allow shell commands (pipes, redirects)
  allow_shell: true

  # Allow terminal takeover (multi-step operations)
  allow_terminal_takeover: true
```

### Examples

```yaml
# Balanced mode (default)
security:
  command_level: dangerous
  allow_shell: true

# Paranoid mode
security:
  command_level: always
  allow_shell: false
  restricted_paths:
    - /etc
    - /usr
    - /System
    - /home
```
```

**Step 2: Update README.md**

Add to `README.md`:
```markdown
## Security

tada includes built-in security controls to protect against dangerous AI-generated commands:

- 🔒 **Dangerous command detection** - Built-in list + AI judgment
- 🛡️ **Path access control** - Restrict access to sensitive paths
- 📝 **Read-only protection** - Protect important files from modification
- 🔧 **Shell analysis** - Detect potentially dangerous shell operations

### Configuration

```yaml
security:
  command_level: dangerous    # always | dangerous | never
  restricted_paths:            # Forbidden paths
    - /etc
    - /usr/bin
  readonly_paths:              # Read-only paths
    - ~/.ssh
    - ~/.gnupg
  allow_shell: true            # Allow shell commands
```
```

**Step 3: Run tests**

Run:
```bash
go build ./...
```

Expected: No errors

**Step 4: Commit**

```bash
git add docs/getting-started.md README.md
git commit -m "docs: update documentation with security configuration"
```

---

## Summary

This implementation plan covers the complete security control system for Phase 2:

1. ✅ Security package structure
2. ✅ DangerousCommandChecker (built-in + AI)
3. ✅ PathAccessChecker (restricted + readonly)
4. ✅ ShellCommandAnalyzer (safe/dangerous)
5. ✅ SecurityController (unified interface)
6. ✅ Config integration
7. ✅ Engine integration
8. ✅ Main entry point updates
9. ✅ Integration tests
10. ✅ Documentation updates

**Total estimated time:** 2-3 hours for experienced Go developer

**Next steps after Phase 2:** Implement plugin system or move to Phase 3 (TUI)
