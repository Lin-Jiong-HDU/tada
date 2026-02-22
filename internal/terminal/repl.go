package terminal

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Lin-Jiong-HDU/tada/internal/conversation"
)

// ErrUserExit 表示用户请求退出
var ErrUserExit = errors.New("user requested exit")

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
			return ErrUserExit
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

	var fullResponse strings.Builder
	for chunk := range stream {
		fullResponse.WriteString(chunk)
	}

	// 清除 "思考中..."
	if r.showThinking {
		fmt.Print("\r\033[K")
	}

	// 渲染美化版本
	fmt.Print("🤖 ")
	if r.renderer != nil {
		rendered, _ := r.renderer.Render(fullResponse.String())
		fmt.Print(rendered)
	} else {
		fmt.Println(fullResponse.String())
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
			// 没有参数，列出可用的 prompts
			r.DisplayAvailablePrompts()
			return false, nil
		}

		// 切换 prompt
		if err := r.manager.SwitchPrompt(r.conversation.ID, parts[1]); err != nil {
			fmt.Printf("切换 prompt 失败: %v\n", err)
			return false, nil
		}

		fmt.Printf("✓ 已切换到 prompt: %s\n", parts[1])
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
  /help              显示此帮助
  /clear             清屏
  /prompt [name]     切换/列出 prompt 模板
  /exit, /quit       退出并保存
`
	fmt.Println(help)
}

// DisplayAvailablePrompts 显示可用的 prompt 模板
func (r *REPL) DisplayAvailablePrompts() {
	prompts, err := r.manager.ListPrompts()
	if err != nil {
		fmt.Printf("获取 prompt 列表失败: %v\n", err)
		return
	}

	if len(prompts) == 0 {
		fmt.Println("没有可用的 prompt 模板")
		return
	}

	fmt.Println("\n可用的 prompt 模板:")
	for _, p := range prompts {
		if p.Title != "" {
			fmt.Printf("  • %s - %s\n", p.Name, p.Title)
		} else {
			fmt.Printf("  • %s\n", p.Name)
		}
		if p.Description != "" {
			fmt.Printf("    %s\n", p.Description)
		}
	}
	fmt.Println()
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
