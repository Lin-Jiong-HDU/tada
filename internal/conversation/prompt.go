package conversation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PromptLoader 加载 prompt 模板
type PromptLoader struct {
	promptsDir string
}

// NewPromptLoader 创建 PromptLoader
func NewPromptLoader(promptsDir string) *PromptLoader {
	return &PromptLoader{
		promptsDir: promptsDir,
	}
}

// PromptTemplate prompt 模板
type PromptTemplate struct {
	Name         string
	Title        string
	Description  string
	Content      string
	SystemPrompt string
}

// Load 加载指定名称的 prompt
func (l *PromptLoader) Load(name string) (*PromptTemplate, error) {
	path := filepath.Join(l.promptsDir, name+".md")

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt: %w", err)
	}

	return l.Parse(string(content)), nil
}

// Parse 解析 prompt 内容
func (l *PromptLoader) Parse(content string) *PromptTemplate {
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		// 没有 frontmatter，整个内容作为 system prompt
		return &PromptTemplate{
			Name:         "default",
			Content:      content,
			SystemPrompt: strings.TrimSpace(content),
		}
	}

	// 解析 frontmatter
	frontmatter := parts[1]
	systemPrompt := strings.TrimSpace(parts[2])

	template := &PromptTemplate{
		Content:      content,
		SystemPrompt: systemPrompt,
	}

	// 解析 frontmatter 中的字段
	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			template.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			template.Name = strings.Trim(template.Name, `"`)
		} else if strings.HasPrefix(line, "title:") {
			template.Title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			template.Title = strings.Trim(template.Title, `"`)
		} else if strings.HasPrefix(line, "description:") {
			template.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			template.Description = strings.Trim(template.Description, `"`)
		}
	}

	if template.Name == "" {
		template.Name = "default"
	}

	return template
}

// List 列出所有可用的 prompt
func (l *PromptLoader) List() ([]*PromptTemplate, error) {
	entries, err := os.ReadDir(l.promptsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompts directory: %w", err)
	}

	var prompts []*PromptTemplate
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		prompt, err := l.Load(name)
		if err != nil {
			continue // 跳过无法加载的文件
		}

		prompts = append(prompts, prompt)
	}

	return prompts, nil
}
