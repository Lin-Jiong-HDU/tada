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
