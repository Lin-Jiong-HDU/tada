package conversation

import (
	"github.com/charmbracelet/glamour"
)

// Renderer Markdown 渲染器
type Renderer struct {
	term *glamour.TermRenderer
}

// NewRenderer 创建 Renderer
func NewRenderer(width int) (*Renderer, error) {
	term, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}

	return &Renderer{term: term}, nil
}

// Render 渲染 markdown
func (r *Renderer) Render(markdown string) (string, error) {
	out, err := r.term.Render(markdown)
	if err != nil {
		// 降级：返回原始文本
		return markdown, nil
	}
	return out, nil
}
