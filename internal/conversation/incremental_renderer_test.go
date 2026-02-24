package conversation

import (
	"testing"
)

func TestNewIncrementalRenderer(t *testing.T) {
	width := 80
	ir, err := NewIncrementalRenderer(width)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ir == nil {
		t.Fatal("Expected renderer to be created")
	}

	if ir.width != width {
		t.Errorf("Expected width %d, got %d", width, ir.width)
	}

	if !ir.isFirst {
		t.Error("Expected isFirst to be true for new renderer")
	}
}

func TestRenderIncremental_FirstRender(t *testing.T) {
	ir, _ := NewIncrementalRenderer(80)

	// First render should render without diff
	markdown := "# Hello\nWorld"
	err := ir.RenderIncremental(markdown)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ir.isFirst {
		t.Error("Expected isFirst to be false after first render")
	}

	if len(ir.oldLines) == 0 {
		t.Error("Expected oldLines to be populated after render")
	}
}

func TestRenderIncremental_DiffRender(t *testing.T) {
	// 测试需要捕获 stdout，这里简化测试 diff 逻辑
	ir, _ := NewIncrementalRenderer(80)

	// First render
	ir.RenderIncremental("# Hello\n")

	// Store old lines
	oldLineCount := ir.lineCount
	oldLinesCopy := make([]string, len(ir.oldLines))
	copy(oldLinesCopy, ir.oldLines)

	// Second render with more content
	ir.RenderIncremental("# Hello\nWorld\n")

	// Verify state was updated
	if ir.lineCount <= oldLineCount {
		t.Errorf("Expected lineCount to increase, was %d now %d", oldLineCount, ir.lineCount)
	}

	if len(ir.oldLines) <= len(oldLinesCopy) {
		t.Error("Expected oldLines to grow")
	}
}
