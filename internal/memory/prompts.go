package memory

import (
	"fmt"
	"os"
	"path/filepath"
)

// PromptTemplates 定义所有内存相关的提示词模板
var PromptTemplates = map[string]string{
	"summary.md": `---
name: "summary"
title: "会话总结"
description: "生成对话摘要用于短期记忆"
---
Summarize the following conversation in 1-2 sentences, focusing on:
- Key topics discussed
- Important decisions or conclusions
- Technical details mentioned (languages, frameworks, tools)
- User preferences or requirements expressed

Keep it concise and factual.`,
	"extract.md": `---
name: "extract"
title: "实体提取"
description: "从对话中提取实体和用户偏好"
---
Extract structured information from this conversation summary.

Please extract and return as JSON:
{
  "entities": ["list of technologies, frameworks, tools mentioned"],
  "preferences": {
    "editor": "preferred editor if mentioned",
    "timezone": "timezone if mentioned",
    "shell": "shell if mentioned",
    "communication_style": "preferred communication style if mentioned"
  },
  "context": ["key topics, projects, or areas of interest discussed"]
}

Only include fields that have values. Return valid JSON only, no markdown.`,
	"system.md": `---
name: "system"
title: "系统提示词"
description: "包含记忆上下文的系统提示词"
---
You are tada, a terminal AI assistant.

## User Profile
{{profile}}

## Recent Conversations
{{summaries}}

Use this context to provide more personalized responses.`,
	"update-profile.md": `---
name: "update-profile"
title: "用户画像更新"
description: "使用LLM更新用户画像"
---
Update the following user profile with the new information provided.

## Current Profile
{{profile}}

## New Information
{{info}}

Please update the profile by:
1. Adding any new information that should be included
2. Updating existing information if the new information contradicts or refines it
3. Removing information that is no longer relevant
4. Keeping the same markdown format

Return ONLY the updated profile markdown, no other text.`,
}

// EnsureDefaultPrompts 确保默认的 memory prompts 存在
func EnsureDefaultPrompts(promptsDir string) error {
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return err
	}

	for name, content := range PromptTemplates {
		path := filepath.Join(promptsDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create prompt %s: %w", name, err)
			}
		}
	}

	return nil
}
