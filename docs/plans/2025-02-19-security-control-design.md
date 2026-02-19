# Security Control Implementation Design

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a comprehensive security control system for tada to protect users from dangerous AI-generated commands.

**Architecture:** A layered security controller that checks commands before execution, combining built-in dangerous command detection with AI judgment.

**Tech Stack:** Go 1.25.7, Viper (config), existing core infrastructure

---

## 1. Overview

This document describes the security control system for Phase 2 of the tada project. The security controller sits between the AI parser and the command executor, protecting users from dangerous operations while maintaining flexibility.

---

## 2. Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      Engine                             │
│  ┌────────────┐      ┌─────────────┐      ┌────────┐  │
│  │ AI Parser  │ ───► │ Security    │ ───► │Executor│  │
│  │            │      │ Controller  │      │        │  │
│  └────────────┘      └─────────────┘      └────────┘  │
│                              │                         │
│                              ▼                         │
│                    ┌─────────────────┐                │
│                    │ Security Policy │                │
│                    │  (from config)  │                │
│                    └─────────────────┘                │
└─────────────────────────────────────────────────────────┘
```

---

## 3. Core Components

### 3.1 SecurityPolicy Configuration

```go
// SecurityPolicy 安全策略配置
type SecurityPolicy struct {
    // 命令确认级别
    CommandLevel ConfirmLevel `mapstructure:"command_level"`

    // 受限路径（禁止访问）
    RestrictedPaths []string `mapstructure:"restricted_paths"`

    // 只读路径（禁止写入）
    ReadOnlyPaths []string `mapstructure:"readonly_paths"`

    // 是否允许 shell 命令
    AllowShell bool `mapstructure:"allow_shell"`

    // 是否允许终端接管（多步操作）
    AllowTerminalTakeover bool `mapstructure:"allow_terminal_takeover"`
}

// ConfirmLevel 命令确认级别
type ConfirmLevel string
const (
    ConfirmAlways     ConfirmLevel = "always"     // 总是确认
    ConfirmDangerous  ConfirmLevel = "dangerous"  // 仅危险命令确认
    ConfirmNever      ConfirmLevel = "never"      // 从不确认
)
```

### 3.2 SecurityController

```go
// SecurityController 安全检查控制器
type SecurityController struct {
    policy              *SecurityPolicy
    dangerChecker       *DangerousCommandChecker
    pathChecker         *PathAccessChecker
    shellAnalyzer       *ShellCommandAnalyzer
}

// CheckResult 检查结果
type CheckResult struct {
    Allowed       bool     // 是否允许执行
    RequiresAuth  bool     // 是否需要授权（Phase 3 TUI）
    Warning       string   // 警告信息
    Reason        string   // 详细原因
}

// CheckCommand 检查命令是否可以执行
func (sc *SecurityController) CheckCommand(cmd ai.Command) (*CheckResult, error)

// CheckPathAccess 检查路径访问权限
func (sc *SecurityController) CheckPathAccess(path string, write bool) (*CheckResult, error)

// AnalyzeShellCommand 分析 shell 命令是否安全
func (sc *SecurityController) AnalyzeShellCommand(cmdStr string) (*CheckResult, error)
```

### 3.3 DangerousCommandChecker

```go
// DangerousCommandChecker 危险命令检测器
type DangerousCommandChecker struct {
    dangerousCommands []string
    dangerousPatterns []string
}

// 内置危险命令
var dangerousCommands = []string{
    "rm", "rmdir", "dd", "mkfs", "format",
    "chmod", "chown", "userdel", "groupdel",
}

// 危险模式
var dangerousPatterns = []string{
    "rm -rf /",
    "rm -rf .*",
    "> /",
    ">: *",
    "chmod 777 /",
}
```

### 3.4 PathAccessChecker

```go
// PathAccessChecker 路径访问检查器
type PathAccessChecker struct {
    restrictedPaths []string
    readOnlyPaths   []string
}

// 检查路径是否受限
func (pc *PathAccessChecker) IsRestricted(path string) bool

// 检查路径是否只读
func (pc *PathAccessChecker) IsReadOnly(path string, write bool) bool

// 解析命令中的路径参数
func (pc *PathAccessChecker) ExtractPaths(cmd ai.Command) []string
```

### 3.5 ShellCommandAnalyzer

```go
// ShellCommandAnalyzer Shell 命令分析器
type ShellCommandAnalyzer struct {
    allowShell bool
}

// 安全的 shell 操作：管道、简单重定向
var safeShellPatterns = []string{"|", ">", ">>"}

// 危险的 shell 操作：重定向到系统路径
var dangerousShellPatterns = []string{
    "> /etc/", "> /usr/", "> /System/",
    "> ~/../", // 尝试访问上级目录
}

// Analyze 分析 shell 命令是否安全
func (sa *ShellCommandAnalyzer) Analyze(cmdStr string) *CheckResult
```

---

## 4. Data Flow

### 4.1 命令执行流程

```
AI 解析意图 → Intent { Commands: [...], NeedsConfirm: bool }
                        │
                        ▼
              ┌─────────────────────┐
              │ SecurityController │
              └─────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
   ┌─────────┐   ┌─────────┐   ┌─────────────┐
   │ 危险命令 │   │路径访问 │   │ Shell 分析  │
   │ 检查    │   │ 检查    │   │             │
   └────┬────┘   └────┬────┘   └──────┬──────┘
        │             │               │
        ▼             ▼               ▼
   CheckResult   CheckResult    CheckResult
        │             │               │
        └──────────────┴───────────────┘
                        │
                        ▼
              ┌─────────────────────┐
              │   汇总检查结果       │
              └─────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
   ┌─────────┐   ┌─────────┐   ┌─────────────┐
   │ 允许执行 │   │ 需要授权 │   │ 拒绝执行    │
   │ (Phase2)│   │(Phase3) │   │ (显示错误)  │
   └─────────┘   └─────────┘   └─────────────┘
```

---

## 5. Configuration

### 5.1 Config Structure Update

```go
// Config holds the application configuration
type Config struct {
    AI       AIConfig       `mapstructure:"ai"`
    Security SecurityPolicy `mapstructure:"security"`
}
```

### 5.2 Default Values

```go
v.SetDefault("security.command_level", "dangerous")
v.SetDefault("security.allow_shell", true)
v.SetDefault("security.allow_terminal_takeover", true)
v.SetDefault("security.restricted_paths", []string{})
v.SetDefault("security.readonly_paths", []string{})
```

### 5.3 Config File Example

```yaml
# ~/.tada/config.yaml
ai:
  provider: glm
  api_key: your-key
  model: glm-5
  base_url: https://open.bigmodel.cn/api

security:
  # 命令确认级别: always | dangerous | never
  command_level: dangerous

  # 禁止访问的路径
  restricted_paths:
    - /etc
    - /usr/bin
    - /System

  # 只读路径（禁止写入）
  readonly_paths:
    - ~/.ssh
    - ~/.gnupg

  # 是否允许 shell 命令
  allow_shell: true

  # 是否允许终端接管（多步操作）
  allow_terminal_takeover: true
```

---

## 6. Error Handling

### 6.1 Error Types

```go
type SecurityError struct {
    Type    SecurityErrorType
    Message string
    Context map[string]interface{}
}

type SecurityErrorType string
const (
    ErrTypeDangerousCommand  SecurityErrorType = "dangerous_command"
    ErrTypeRestrictedPath    SecurityErrorType = "restricted_path"
    ErrTypeReadOnlyPath      SecurityErrorType = "readonly_path"
    ErrTypeDangerousShell    SecurityErrorType = "dangerous_shell"
    ErrTypeShellDisabled     SecurityErrorType = "shell_disabled"
)
```

### 6.2 User-Friendly Messages

| Scenario | Message | Action |
|----------|---------|--------|
| Dangerous command | `⚠️  危险命令: rm -rf / 可能删除系统文件` | Requires confirmation |
| Restricted path | `🚫 拒绝访问: /etc 受系统保护` | Reject |
| Read-only write | `⚠️  只读保护: ~/.ssh 不允许写入` | Requires confirmation |
| Shell disabled | `⚠️  Shell 命令已禁用（allow_shell=false）` | Reject |
| Dangerous shell | `⚠️  危险操作: 尝试写入 /etc/ 需要授权` | Requires TUI auth |

### 6.3 Phase 2 Temporary Handling

```go
switch {
case result.Allowed:
    return nil

case result.RequiresAuth:
    fmt.Printf("⚠️  %s\n", result.Warning)
    fmt.Println("⚠️  注意: 完整的授权确认将在 Phase 3 (TUI) 中实现")
    // Auto-continue for now

default:
    return fmt.Errorf("🚫 安全拒绝: %s", result.Reason)
}
```

---

## 7. Testing Strategy

### 7.1 Test Structure

```
tests/
├── unit/
│   └── security/
│       ├── controller_test.go        # SecurityController 单元测试
│       ├── danger_checker_test.go    # 危险命令检测测试
│       ├── path_checker_test.go      # 路径检查测试
│       └── shell_analyzer_test.go    # Shell 分析测试
│
├── integration/
│   └── security/
│       ├── engine_integration_test.go # Engine 集成测试
│       └── config_test.go             # 配置集成测试
│
└── e2e/
    └── security/
        └── scenarios_test.go          # 真实场景测试
```

### 7.2 Key Test Scenarios

**Dangerous Command Detection:**
- [x] `rm -rf /` is marked as dangerous
- [x] `ls` is not marked as dangerous
- [x] `chmod 777 ~/.ssh` is marked as dangerous
- [x] AI-judged dangerous commands are correctly identified

**Path Access Control:**
- [x] Accessing `/etc/passwd` is rejected
- [x] Writing to `~/.ssh/id_rsa` shows warning
- [x] Normal file operations are unaffected

**Shell Command Analysis:**
- [x] `ls | grep test` is identified as safe
- [x] `cat file > /etc/config` is identified as dangerous
- [x] All shell commands rejected when `allow_shell=false`

**Edge Cases:**
- [x] Empty command handling
- [x] Path traversal attacks (`../../../etc/passwd`)
- [x] Symbolic link handling
- [x] Environment variable expansion

---

## 8. Implementation Plan

### Phase 2.1: Core Security Control

**Files to create:**
```
internal/core/security/
├── policy.go           # SecurityPolicy structure
├── controller.go       # SecurityController
├── danger_checker.go   # DangerousCommandChecker
├── path_checker.go     # PathAccessChecker
└── shell_analyzer.go   # ShellCommandAnalyzer
```

**Tasks:**
1. Define SecurityPolicy structure
2. Implement DangerousCommandChecker
3. Implement PathAccessChecker
4. Implement ShellCommandAnalyzer
5. Implement SecurityController
6. Integrate into Engine.Process()
7. Update config loading
8. Write unit tests

### Phase 2.2: Terminal Takeover (Optional)

**Tasks:**
1. Implement multi-step operation mode
2. Implement manual intervention mode
3. Write tests

---

## 9. Integration Points

### Files to Modify

| File | Changes |
|------|---------|
| `internal/storage/config.go` | Add SecurityPolicy field |
| `internal/core/engine.go` | Integrate SecurityController |
| `cmd/tada/main.go` | Pass SecurityPolicy to Engine |

### Files to Create

| File | Purpose |
|------|---------|
| `internal/core/security/policy.go` | Policy definition |
| `internal/core/security/controller.go` | Controller logic |
| `internal/core/security/danger_checker.go` | Danger detection |
| `internal/core/security/path_checker.go` | Path checking |
| `internal/core/security/shell_analyzer.go` | Shell analysis |
| `internal/core/security/*_test.go` | Tests |

---

## 10. Success Criteria

- [ ] All security checks pass before command execution
- [ ] Dangerous commands are detected (both built-in and AI-judged)
- [ ] Restricted paths are properly blocked
- [ ] Read-only paths show warnings for write operations
- [ ] Shell commands are analyzed for safety
- [ ] Configuration defaults work as expected (balanced mode)
- [ ] Unit tests cover all major scenarios
- [ ] Integration tests verify end-to-end security
