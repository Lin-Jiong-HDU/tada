package conversation

import (
	"fmt"
	"strings"
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

	// Diff: 找到第一个不同的行
	diffIndex := findDiffIndex(ir.oldLines, newLines)

	if diffIndex == -1 {
		// 内容完全相同，不需要重绘
		return nil
	}

	// 计算需要向上移动的行数
	moveUp := ir.lineCount - diffIndex

	// 光标回退到差异行
	fmt.Printf("\033[%dA", moveUp)

	// 清除之前渲染的行（只清除到之前渲染的最后行，不是整个屏幕）
	linesToClear := ir.lineCount - diffIndex
	for i := 0; i < linesToClear; i++ {
		fmt.Print("\033[K") // 清除当前行到行尾
		if i < linesToClear-1 {
			fmt.Print("\n") // 移动到下一行（最后一行不需要移动）
		}
	}

	// 从差异行开始重绘
	for i := diffIndex; i < len(newLines); i++ {
		if i == len(newLines)-1 {
			// 最后一行不需要换行（避免额外空行）
			fmt.Print(newLines[i])
		} else {
			fmt.Println(newLines[i])
		}
	}

	// 更新状态
	ir.oldLines = newLines
	ir.lineCount = len(newLines)

	return nil
}

// Reset 重置渲染器状态 (用于 resize 后)
func (ir *IncrementalRenderer) Reset() {
	ir.oldLines = nil
	ir.lineCount = 0
	ir.isFirst = true
}

// SetWidth 更新终端宽度 (用于 resize)
// 宽度变化后需要重新创建基础渲染器
func (ir *IncrementalRenderer) SetWidth(width int) error {
	baseRenderer, err := NewRenderer(width)
	if err != nil {
		return err
	}

	ir.baseRenderer = baseRenderer
	ir.width = width
	return nil
}

// LineCount 返回当前渲染的行数（用于测试）
func (ir *IncrementalRenderer) LineCount() int {
	return ir.lineCount
}

// splitLines 按行切分字符串
func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

// findDiffIndex 找到两个切片的第一个差异索引
func findDiffIndex(oldLines, newLines []string) int {
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	for i := 0; i < maxLen; i++ {
		if i >= len(oldLines) || i >= len(newLines) {
			return i
		}
		if oldLines[i] != newLines[i] {
			return i
		}
	}

	return -1 // 完全相同
}
