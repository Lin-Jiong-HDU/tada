package terminal

import (
	"fmt"
	"strings"

	"github.com/Lin-Jiong-HDU/tada/internal/conversation"
)

// REPL 交互式对话
type REPL struct {
	manager      *conversation.Manager
	conversation *conversation.Conversation
	renderer     *conversation.Renderer
	stream       bool
	showThinking bool
}

// NewREPL 创建 REPL
func NewREPL(manager *conversation.Manager, conv *conversation.Conversation, stream bool) *REPL {
	return &REPL{
		manager:      manager,
		conversation: conv,
		stream:       stream,
		showThinking: true,
	}
}

// SetRenderer 设置渲染器
func (r *REPL) SetRenderer(renderer *conversation.Renderer) {
	r.renderer = renderer
}

// ProcessInput 处理用户输入
func (r *REPL) ProcessInput(input string) error {
	input = strings.TrimSpace(input)

	// 检查是否是命令
	if strings.HasPrefix(input, "/") {
		shouldExit, err := r.HandleCommand(input)
		if err != nil {
			return err
		}
		if shouldExit {
			return fmt.Errorf("exit")
		}
		return nil
	}

	// 普通对话
	if r.stream {
		return r.processStreamChat(input)
	}

	return r.processChat(input)
}

// processChat 处理普通对话
func (r *REPL) processChat(input string) error {
	response, err := r.manager.Chat(r.conversation.ID, input)
	if err != nil {
		return err
	}

	// 渲染 markdown
	if r.renderer != nil {
		rendered, _ := r.renderer.Render(response)
		fmt.Print(rendered)
	} else {
		fmt.Println(response)
	}

	return nil
}

// processStreamChat 处理流式对话
func (r *REPL) processStreamChat(input string) error {
	if r.showThinking {
		fmt.Print("🤠 思考中...")
	}

	stream, err := r.manager.ChatStream(r.conversation.ID, input)
	if err != nil {
		return err
	}

	// 清除 "思考中..."
	if r.showThinking {
		fmt.Print("\r\033[K")
	}

	fmt.Print("🤖 ")

	var fullResponse strings.Builder
	for chunk := range stream {
		fmt.Print(chunk)
		fullResponse.WriteString(chunk)
	}

	fmt.Println()

	// 重新渲染美化版本
	if r.renderer != nil {
		rendered, _ := r.renderer.Render(fullResponse.String())
		fmt.Print(rendered)
	}

	return nil
}

// HandleCommand 处理命令
func (r *REPL) HandleCommand(cmd string) (bool, error) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false, nil
	}

	switch parts[0] {
	case "/exit", "/quit":
		r.DisplayExitSummary()
		return true, nil

	case "/help":
		r.DisplayHelp()
		return false, nil

	case "/clear":
		fmt.Print("\033[H\033[2J") // ANSI 清屏
		return false, nil

	case "/prompt":
		if len(parts) < 2 {
			fmt.Println("用法: /prompt <name>")
			return false, nil
		}
		fmt.Printf("切换 prompt: %s (未实现)\n", parts[1])
		return false, nil

	default:
		fmt.Printf("未知命令: %s\n", parts[0])
		return false, nil
	}
}

// DisplayHelp 显示帮助
func (r *REPL) DisplayHelp() {
	help := `
可用命令:
  /help         显示此帮助
  /clear        清屏
  /prompt <name> 切换 prompt 模板
  /exit, /quit  退出并保存
`
	fmt.Println(help)
}

// DisplayExitSummary 显示退出摘要
func (r *REPL) DisplayExitSummary() {
	fmt.Println("📝 对话已保存")
	fmt.Printf("   ID: %s\n", r.conversation.ID)
	fmt.Printf("   消息: %d 条\n", len(r.conversation.Messages))
	fmt.Printf("   恢复: tada chat --continue %s\n", r.conversation.ID)
}

// Run 运行 REPL 主循环
func (r *REPL) Run() error {
	// 注意：这是一个简化的实现
	// 实际的 CLI 交互会在 chatCmd 中处理
	return nil
}
