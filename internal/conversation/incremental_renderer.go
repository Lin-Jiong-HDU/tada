package conversation

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
