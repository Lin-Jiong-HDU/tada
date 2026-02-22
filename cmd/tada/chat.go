package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
	"github.com/Lin-Jiong-HDU/tada/internal/ai/glm"
	"github.com/Lin-Jiong-HDU/tada/internal/ai/openai"
	"github.com/Lin-Jiong-HDU/tada/internal/conversation"
	"github.com/Lin-Jiong-HDU/tada/internal/storage"
	"github.com/Lin-Jiong-HDU/tada/internal/terminal"
	"github.com/spf13/cobra"
)

var (
	chatPromptName string
	chatContinueID string
	chatList       bool
	chatToday      bool
	chatShowID     string
	chatDeleteID   string
	chatName       string
	chatNoHistory  bool
	chatNoStream   bool
	chatNoRender   bool
)

func getChatCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "与 AI 对话",
		Long:  "交互式 AI 对话，支持多轮对话、历史记录和自定义 prompt",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			_, err := storage.InitConfig()
			return err
		},
		RunE: runChat,
	}

	cmd.Flags().StringVarP(&chatPromptName, "prompt", "p", "default", "Prompt 模板名称")
	cmd.Flags().StringVarP(&chatContinueID, "continue", "c", "", "恢复对话 ID")
	cmd.Flags().BoolVarP(&chatList, "list", "l", false, "列出所有对话")
	cmd.Flags().BoolVar(&chatToday, "today", false, "仅列出今天的对话")
	cmd.Flags().StringVarP(&chatShowID, "show", "s", "", "显示对话详情")
	cmd.Flags().StringVarP(&chatDeleteID, "delete", "d", "", "删除对话")
	cmd.Flags().StringVarP(&chatName, "name", "n", "", "对话名称")
	cmd.Flags().BoolVar(&chatNoHistory, "no-history", false, "不保存历史")
	cmd.Flags().BoolVar(&chatNoStream, "no-stream", false, "禁用流式输出")
	cmd.Flags().BoolVar(&chatNoRender, "no-render", false, "禁用 markdown 渲染")

	return cmd
}

func runChat(cmd *cobra.Command, args []string) error {
	cfg := storage.GetConfig()

	// 验证 API key
	if cfg.AI.APIKey == "" {
		return fmt.Errorf("AI API key 未配置，请在 ~/.tada/config.yaml 中设置")
	}

	// 创建 AI provider
	var aiProvider ai.AIProvider
	switch cfg.AI.Provider {
	case "openai":
		aiProvider = openai.NewClient(cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL)
	case "glm", "zhipu":
		aiProvider = glm.NewClient(cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL)
	default:
		return fmt.Errorf("不支持的 provider: %s", cfg.AI.Provider)
	}

	// 初始化存储
	configDir, _ := storage.GetConfigDir()
	conversationsDir := filepath.Join(configDir, "conversations")
	promptsDir := filepath.Join(configDir, "prompts")

	// 确保默认 prompts 存在
	if err := conversation.EnsureDefaultPrompts(promptsDir); err != nil {
		return fmt.Errorf("初始化 prompts 失败: %w", err)
	}

	convStorage := conversation.NewFileStorage(conversationsDir)
	promptLoader := conversation.NewPromptLoader(promptsDir)
	manager := conversation.NewManager(convStorage, promptLoader, aiProvider)

	// 处理子命令
	if chatList {
		return runListConversations(manager)
	}

	if chatShowID != "" {
		return runShowConversation(manager, chatShowID)
	}

	if chatDeleteID != "" {
		return runDeleteConversation(manager, chatDeleteID)
	}

	// 创建或恢复对话
	var conv *conversation.Conversation
	var err error

	if chatContinueID != "" {
		conv, err = manager.Get(chatContinueID)
		if err != nil {
			return fmt.Errorf("对话不存在: %s", chatContinueID)
		}
		fmt.Printf("📂 恢复对话: %s (%s)\n", conv.ID, conv.PromptName)
	} else if chatNoHistory {
		// 使用临时对话，不保存历史
		conv, err = manager.CreateEphemeral(chatName, chatPromptName)
		if err != nil {
			return fmt.Errorf("创建临时对话失败: %w", err)
		}
		fmt.Printf("📝 临时对话 (%s) - 不保存历史\n", conv.PromptName)
	} else {
		conv, err = manager.Create(chatName, chatPromptName)
		if err != nil {
			return fmt.Errorf("创建对话失败: %w", err)
		}
		fmt.Printf("📝 新对话 (%s)\n", conv.PromptName)
	}

	// 创建 renderer
	var renderer *conversation.Renderer
	if !chatNoRender {
		renderer, _ = conversation.NewRenderer(80)
	}

	// 运行 REPL
	repl := terminal.NewREPL(manager, conv, !chatNoStream)
	repl.SetRenderer(renderer)

	fmt.Println("💬 输入消息，/help 查看命令，/exit 退出")
	fmt.Println()

	return runREPLLoop(repl)
}

// runREPLLoop 运行 REPL 交互循环
func runREPLLoop(repl *terminal.REPL) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		// 显示提示符
		fmt.Print("👉 ")

		// 读取输入
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("读取输入失败: %w", err)
		}

		// 去除换行符和空格
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// 处理输入
		err = repl.ProcessInput(input)
		if err != nil {
			// 检查是否是退出信号
			if errors.Is(err, terminal.ErrUserExit) {
				return nil
			}
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		}

		fmt.Println()
	}
}

func runListConversations(manager *conversation.Manager) error {
	convs, err := manager.List()
	if err != nil {
		return err
	}

	if len(convs) == 0 {
		fmt.Println("💬 没有对话记录")
		return nil
	}

	fmt.Println("💬 对话历史:")
	fmt.Println()

	for _, conv := range convs {
		fmt.Printf("  %s  [%s]  %d 条消息  %s\n",
			conv.ID[:12],
			conv.PromptName,
			len(conv.Messages),
			conv.UpdatedAt.Format("2006-01-02 15:04"),
		)
	}

	return nil
}

func runShowConversation(manager *conversation.Manager, id string) error {
	conv, err := manager.Get(id)
	if err != nil {
		return fmt.Errorf("对话不存在: %w", err)
	}

	fmt.Printf("对话: %s\n", conv.ID)
	fmt.Printf("Prompt: %s\n", conv.PromptName)
	fmt.Printf("创建时间: %s\n", conv.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("消息数: %d\n", len(conv.Messages))
	fmt.Println("\n消息:")
	fmt.Println()

	for _, msg := range conv.Messages {
		fmt.Printf("[%s]: %s\n\n", msg.Role, msg.Content)
	}

	return nil
}

func runDeleteConversation(manager *conversation.Manager, id string) error {
	err := manager.Delete(id)
	if err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}

	fmt.Printf("✓ 对话已删除: %s\n", id)
	return nil
}
