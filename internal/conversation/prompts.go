package conversation

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureDefaultPrompts 确保默认 prompt 存在
func EnsureDefaultPrompts(promptsDir string) error {
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return err
	}

	prompts := map[string]string{
		"default.md": `---
name: "default"
title: "默认助手"
description: "友好的 AI 助手"
---

你是一个友好、乐于助人的 AI 助手。请用简洁、准确的方式回答用户的问题。`,
		"coder.md": `---
name: "coder"
title: "编程助手"
description: "专业的编程对话助手"
---

你是一位经验丰富的程序员，擅长 Go、Python、JavaScript、TypeScript 等编程语言。

你的回答应该：
- 简洁、准确
- 提供可执行的代码示例
- 解释代码的工作原理
- 遵循最佳实践`,
		"expert.md": `---
name: "expert"
title: "技术专家"
description: "深入的技术分析和解答"
---

你是一位技术专家，能够提供深入的技术分析和解答。

你的回答应该：
- 深入分析问题的本质
- 提供多种解决方案
- 讨论各种方案的权衡
- 给出专业建议`,
	}

	for name, content := range prompts {
		path := filepath.Join(promptsDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create prompt %s: %w", name, err)
			}
		}
	}

	return nil
}
