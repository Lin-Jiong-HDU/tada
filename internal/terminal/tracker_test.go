package terminal

import (
	"testing"
)

func TestNewLineTracker(t *testing.T) {
	maxLines := 10
	tracker, err := NewLineTracker(maxLines)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if tracker == nil {
		t.Fatal("Expected tracker to be created")
	}

	if tracker.maxLines != maxLines {
		t.Errorf("Expected maxLines %d, got %d", maxLines, tracker.maxLines)
	}

	if tracker.stopped {
		t.Error("Expected stopped to be false for new tracker")
	}

	if tracker.lineCount != 1 {
		t.Errorf("Expected initial lineCount 1, got %d", tracker.lineCount)
	}
}