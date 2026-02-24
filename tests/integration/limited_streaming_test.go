package integration

import (
	"strings"
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/terminal"
)

func TestLineTracker_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("LimitedMode", func(t *testing.T) {
		tracker, err := terminal.NewLineTracker(5)
		if err != nil {
			t.Fatalf("Failed to create tracker: %v", err)
		}

		chunks := []string{
			"Line 1\n",
			"Line 2\n",
			"Line 3\n",
			"Line 4\n",
			"Line 5\n",
			"Line 6\n",
		}

		for i, chunk := range chunks {
			display, overflow := tracker.Track(chunk)
			t.Logf("Chunk %d: display='%s', overflow=%v", i, display, overflow)

			// Overflow should start at chunk 4 (Line 5), since lineCount starts at 1
			if i < 4 && overflow {
				t.Errorf("Chunk %d: unexpected overflow", i)
			}
			if i >= 4 && !overflow && display != "" {
				t.Errorf("Chunk %d: expected overflow", i)
			}
		}

		t.Logf("Final lineCount: %d", tracker.LineCount())
	})

	t.Run("UnlimitedMode", func(t *testing.T) {
		tracker, err := terminal.NewLineTracker(0)
		if err != nil {
			t.Fatalf("Failed to create tracker: %v", err)
		}

		longText := strings.Repeat("This is a line\n", 100)
		display, overflow := tracker.Track(longText)

		if overflow {
			t.Error("Expected no overflow in unlimited mode")
		}

		if display != longText {
			t.Error("Expected full text in unlimited mode")
		}
	})
}