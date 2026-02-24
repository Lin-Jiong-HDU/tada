package integration

import (
	"os"
	"strings"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/conversation"
)

func TestIncrementalRendering_Integration(t *testing.T) {
	if os.Getenv("TADA_INTEGRATION_TEST") == "" {
		t.Skip("Set TADA_INTEGRATION_TEST=1")
	}

	ir, err := conversation.NewIncrementalRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	// 模拟流式输入
	chunks := []string{
		"# Hello\n",
		"## ",
		"World\n",
		"```",
		"go\n",
		"fmt.Println",
		"(\"hi\")\n",
		"```\n",
	}

	var full strings.Builder
	for _, chunk := range chunks {
		full.WriteString(chunk)
		err := ir.RenderIncremental(full.String())
		if err != nil {
			t.Errorf("RenderIncremental failed: %v", err)
		}
	}

	// 验证最终状态
	if ir.LineCount() == 0 {
		t.Error("Expected non-zero line count")
	}
}
