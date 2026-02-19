# TUI 界面设计文档

## 概述

本文档描述 tada 的 TUI 界面设计，主要包括：
1. 同步命令的终端确认输入
2. 异步命令的授权队列管理

## 设计目标

1. **同步命令**：使用简单的终端输入进行确认
2. **异步命令**：使用 TUI 界面管理授权队列
3. **持久化**：队列数据存储在会话目录的 JSON 文件中

## 架构

### 同步命令流程

```
Engine.Process()
  └─> securityController.CheckCommand()
      └─> RequiresAuth && !IsAsync?
          ├─ 是 → 终端确认输入 (y/s/q)
          │   ├─ y → 执行命令
          │   ├─ s → 跳过命令
          │   └─ q → 取消全部
          └─ 否 → 直接执行
```

### 异步命令流程

```
Engine.Process()
  └─> securityController.CheckCommand()
      └─> RequiresAuth && IsAsync?
          ├─ 是 → 加入队列 (状态: pending)
          └─ 否 → 直接执行

用户运行 tada tasks
  └─> 加载所有会话的 queue.json
      └─> TUI 显示待授权任务
          ├─ 授权任务 (a) → 立即后台执行
          ├─ 拒绝任务 (r) → 状态变更为 rejected
          └─ 退出 (q) → 已授权任务继续执行，结果记录到文件
```

## 文件结构

```
internal/
├── core/
│   ├── tui/
│   │   ├── queue_model.go       # TUI Model (Bubble Tea)
│   │   ├── queue_view.go        # 界面渲染
│   │   ├── types.go             # TUI 类型定义
│   │   └── keys.go              # 按键绑定
│   └── queue/
│       ├── queue.go             # 队列管理器
│       ├── task.go              # 任务结构和状态
│       └── store.go             # JSON 持久化
├── terminal/
│   └── prompt.go                # 终端确认输入
cmd/tada/
├── main.go                      # 添加 tasks 命令
└── tasks.go                     # tada tasks 命令实现
```

## 数据结构

### Task 状态

```go
type TaskStatus string

const (
    TaskStatusPending   TaskStatus = "pending"    // 待授权
    TaskStatusApproved  TaskStatus = "approved"   // 已授权
    TaskStatusRejected  TaskStatus = "rejected"   // 已拒绝
    TaskStatusExecuting TaskStatus = "executing"  // 执行中
    TaskStatusCompleted TaskStatus = "completed"  // 已完成
    TaskStatusFailed    TaskStatus = "failed"     // 执行失败
)
```

### Task 结构

```go
type Task struct {
    ID          string            `json:"id"`
    SessionID   string            `json:"session_id"`
    Command     ai.Command        `json:"command"`
    CheckResult *security.CheckResult `json:"check_result"`
    Status      TaskStatus        `json:"status"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
    Result      *ExecutionResult  `json:"result,omitempty"`
}

type ExecutionResult struct {
    ExitCode int    `json:"exit_code"`
    Output   string `json:"output"`
    Error    string `json:"error,omitempty"`
}
```

### 队列文件结构

```
~/.tada/sessions/{session-id}/queue.json
```

```json
{
  "tasks": [
    {
      "id": "uuid-1",
      "session_id": "2025-02-19-143022",
      "command": {
        "cmd": "rm",
        "args": ["-rf", "/tmp/test"]
      },
      "check_result": {
        "allowed": true,
        "requires_auth": true,
        "warning": "Dangerous command: rm -rf /tmp/test",
        "reason": "Command is in the dangerous list"
      },
      "status": "pending",
      "created_at": "2025-02-19T10:00:00Z",
      "updated_at": "2025-02-19T10:00:00Z"
    }
  ]
}
```

## TUI 界面设计

### 界面布局

```
┌─────────────────────────────────────────────────────────────┐
│                     tada 任务队列                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  会话: 2025-02-19-143022                                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ [ ] rm -rf /tmp/test                                │   │
│  │     警告: Dangerous command                          │   │
│  │     原因: Command is in the dangerous list          │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  会话: 2025-02-19-150145                                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ [*] dd if=/dev/zero of=file                         │   │
│  │     警告: Dangerous command                          │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  [a] 授权选中  [r] 拒绝选中  [A] 全部授权  [q] 退出          │
└─────────────────────────────────────────────────────────────┘
```

### 按键绑定

| 按键 | 功能 |
|------|------|
| `↑` / `↓` | 选择任务 |
| `a` | 授权选中的任务 |
| `r` | 拒绝选中的任务 |
| `A` | 授权所有待授权任务 |
| `R` | 拒绝所有待授权任务 |
| `Enter` | 查看任务详情 |
| `q` / `ESC` | 退出 |

### 标准功能集

1. 查看待授权任务（按会话分组）
2. 授权/拒绝单个任务
3. 批量操作（全部授权/全部拒绝）
4. 查看任务详情（命令、警告、原因）
5. 退出

## 终端确认输入

### 交互方式

```
⚠️  此操作需要您的授权

命令: rm -rf /tmp/test
警告: Dangerous command: rm -rf /tmp/test
原因: Command is in the dangerous list

[y] 执行  [s] 跳过  [q] 取消全部
> _
```

### 输入验证

- 循环等待直到输入有效选项
- 支持大小写（y/Y, s/S, q/Q）
- 超时选项（可选）

## 命令接口

### tada tasks

打开任务队列 TUI 界面。

```bash
tada tasks
```

行为：
1. 扫描 `~/.tada/sessions/` 目录
2. 加载所有会话的 `queue.json` 文件
3. 按 session_id 分组显示待授权任务
4. 用户进行授权/拒绝操作
5. 退出时已授权任务继续执行

## 与现有组件的集成

### Engine.Process 修改

```go
func (e *Engine) Process(ctx context.Context, input string, systemPrompt string) error {
    // ... 现有代码 ...

    for i, cmd := range intent.Commands {
        result, err := e.securityController.CheckCommand(cmd)

        if !result.Allowed {
            continue
        }

        if result.RequiresAuth {
            if cmd.IsAsync {
                // 添加到队列
                e.queue.AddTask(cmd, result)
            } else {
                // 终端确认
                confirmed := terminal.Confirm(cmd, result)
                if !confirmed {
                    continue
                }
            }
        }

        e.executor.Execute(ctx, cmd)
    }

    return nil
}
```

### ai.Command 扩展

需要添加 `IsAsync` 字段来标识命令是否异步执行：

```go
type Command struct {
    Cmd    string   `json:"cmd"`
    Args   []string `json:"args"`
    IsAsync bool    `json:"is_async"` // 新增
}
```

## 测试策略

1. **单元测试**
   - 队列管理器测试
   - 任务状态转换测试
   - JSON 持久化测试
   - 终端输入测试

2. **集成测试**
   - TUI 交互测试
   - 队列与 Engine 集成测试

3. **E2E 测试**
   - 完整的授权流程测试

## 后续扩展

1. **完整功能集**（Phase 4+）
   - 直接在 TUI 中执行任务
   - 查看已执行任务的结果
   - 删除已完成的任务
   - 清空队列

2. **插件系统集成**
   - 插件可以注册自己的任务类型
   - 自定义任务状态和操作
