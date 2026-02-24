package conversation

import (
	"fmt"
)

// IncrementalRenderer 增量渲染器
type IncrementalRenderer struct {
	baseRenderer *Renderer // 基础 Glamour 渲染器
	width        int       // 终端宽度
	oldLines     []string  // 上次渲染的行
	lineCount    int       // 上次渲染的总行数
	isFirst      bool      // 是否首次渲染
}

// NewIncrementalRenderer 创建增量渲染器
func NewIncrementalRenderer(width int) (*IncrementalRenderer, error) {
	baseRenderer, err := NewRenderer(width)
	if err != nil {
		return nil, err
	}

	return &IncrementalRenderer{
		baseRenderer: baseRenderer,
		width:        width,
		oldLines:     nil,
		lineCount:    0,
		isFirst:      true,
	}, nil
}

// RenderIncremental 增量渲染 markdown
func (ir *IncrementalRenderer) RenderIncremental(markdown string) error {
	// 使用基础渲染器渲染完整 markdown
	rendered, err := ir.baseRenderer.Render(markdown)
	if err != nil {
		return err
	}

	// 按行切分
	newLines := splitLines(rendered)

	if ir.isFirst {
		// 首次渲染：直接输出所有内容
		for _, line := range newLines {
			fmt.Println(line)
		}
		ir.isFirst = false
		ir.oldLines = newLines
		ir.lineCount = len(newLines)
		return nil
	}

	// Diff 逻辑在下一个任务实现
	return nil
}

// splitLines 按行切分字符串
func splitLines(s string) []string {
	lines := make([]string, 0)
	current := ""

	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}

	// 添加最后一行（可能没有换行符）
	if current != "" || len(lines) == 0 {
		lines = append(lines, current)
	}

	return lines
}
