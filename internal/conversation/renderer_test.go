package conversation

import (
	"testing"
)

func TestRenderer_Render(t *testing.T) {
	renderer, err := NewRenderer(80)
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	// 测试 markdown 渲染
	markdown := "# Hello\n\nThis is **bold** and *italic*.\n\n```go\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n```\n"

	rendered, err := renderer.Render(markdown)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if rendered == "" {
		t.Error("Expected non-empty rendered output")
	}

	// 渲染后的文本应该包含 ANSI 颜色代码
	// glamour 使用 lipgloss 添加颜色
	if rendered == markdown {
		t.Log("Warning: Rendered output same as input (glamour may not be working)")
	}
}

func TestRenderer_RenderPlainText(t *testing.T) {
	renderer, _ := NewRenderer(80)

	text := "Plain text without markdown"

	rendered, err := renderer.Render(text)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if rendered != text {
		t.Logf("Plain text was modified: %s", rendered)
	}
}
