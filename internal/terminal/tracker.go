package terminal

import (
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// LineTracker 追踪流式输出的行数
type LineTracker struct {
	maxWidth   int  // 终端宽度
	currentPos int  // 当前行内的字符位置
	lineCount  int  // 总行数（包括自动换行）
	maxLines   int  // 最大行数限制
	stopped    bool // 是否已停止显示（超限后）
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

// Track 追踪文本，返回应该显示的文本和是否超限
func (t *LineTracker) Track(text string) (displayText string, overflow bool) {
	if t.stopped {
		return "", false
	}

	// 无限制模式
	if t.maxLines == 0 {
		return text, false
	}

	var result strings.Builder

	for _, r := range text {
		// 处理换行符
		if r == '\n' {
			t.lineCount++
			t.currentPos = 0
			result.WriteRune(r)
			if t.lineCount > t.maxLines {
				t.stopped = true
				return result.String(), true
			}
			continue
		}

		charWidth := runewidth.RuneWidth(r)

		// 检查是否需要自动换行
		if t.currentPos+charWidth > t.maxWidth {
			t.lineCount++
			t.currentPos = 0
			if t.lineCount > t.maxLines {
				t.stopped = true
				return result.String(), true
			}
		}

		t.currentPos += charWidth
		result.WriteRune(r)
	}

	return result.String(), false
}

// LineCount 返回当前追踪的总行数
func (t *LineTracker) LineCount() int {
	return t.lineCount
}

// Reset 重置追踪器状态
func (t *LineTracker) Reset() {
	t.currentPos = 0
	t.lineCount = 1
	t.stopped = false
}
