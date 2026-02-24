package terminal

import (
	"os"

	"golang.org/x/term"
)

// LineTracker 追踪流式输出的行数
type LineTracker struct {
	maxWidth    int     // 终端宽度
	currentPos  int     // 当前行内的字符位置
	lineCount   int     // 总行数（包括自动换行）
	maxLines    int     // 最大行数限制
	stopped     bool    // 是否已停止显示（超限后）
}

// NewLineTracker 创建行数追踪器
func NewLineTracker(maxLines int) (*LineTracker, error) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80 // 默认宽度
	}

	return &LineTracker{
		maxWidth:   width,
		currentPos: 0,
		lineCount:  1, // 从第1行开始
		maxLines:   maxLines,
		stopped:    false,
	}, nil
}