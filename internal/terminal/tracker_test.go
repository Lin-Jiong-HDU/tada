package terminal

import (
	"strings"
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

func TestLineTracker_SimpleText(t *testing.T) {
	tracker, _ := NewLineTracker(10)
	display, overflow := tracker.Track("Hello")
	if overflow {
		t.Error("Expected no overflow for short text")
	}
	if display != "Hello" {
		t.Errorf("Expected display 'Hello', got '%s'", display)
	}
	if tracker.lineCount != 1 {
		t.Errorf("Expected lineCount 1, got %d", tracker.lineCount)
	}
}

func TestLineTracker_WithNewlines(t *testing.T) {
	tracker, _ := NewLineTracker(10)
	display, overflow := tracker.Track("Line 1\nLine 2\nLine 3")
	if overflow {
		t.Error("Expected no overflow for 3 lines")
	}
	if display != "Line 1\nLine 2\nLine 3" {
		t.Errorf("Unexpected display: '%s'", display)
	}
	if tracker.lineCount != 3 {
		t.Errorf("Expected lineCount 3, got %d", tracker.lineCount)
	}
}

func TestLineTracker_TrailingNewline(t *testing.T) {
	tracker, _ := NewLineTracker(10)
	display, overflow := tracker.Track("Hello\n")
	if overflow {
		t.Error("Expected no overflow")
	}
	if tracker.lineCount != 2 {
		t.Errorf("Expected lineCount 2, got %d", tracker.lineCount)
	}
	_ = display // use display
}

func TestLineTracker_EmptyString(t *testing.T) {
	tracker, _ := NewLineTracker(10)
	display, overflow := tracker.Track("")
	if overflow {
		t.Error("Expected no overflow for empty string")
	}
	if display != "" {
		t.Errorf("Expected empty display, got '%s'", display)
	}
}

func TestLineTracker_Overflow(t *testing.T) {
	tracker, _ := NewLineTracker(2)
	display1, overflow1 := tracker.Track("Line 1\n")
	if overflow1 {
		t.Error("Expected no overflow on first line")
	}
	display2, overflow2 := tracker.Track("Line 2\nLine 3\n")
	if !overflow2 {
		t.Error("Expected overflow on third line")
	}
	if display2 != "Line 2\n" {
		t.Errorf("Expected display to stop at limit, got '%s'", display2)
	}
	_ = display1
}

func TestLineTracker_LineCount(t *testing.T) {
	tracker, _ := NewLineTracker(10)
	tracker.Track("Line 1\nLine 2\n")
	if tracker.LineCount() != 3 {
		t.Errorf("Expected LineCount 3, got %d", tracker.LineCount())
	}
}

func TestLineTracker_Reset(t *testing.T) {
	tracker, _ := NewLineTracker(10)
	tracker.Track("Line 1\nLine 2\n")
	tracker.Reset()
	if tracker.lineCount != 1 {
		t.Errorf("Expected lineCount 1 after reset, got %d", tracker.lineCount)
	}
	if tracker.currentPos != 0 {
		t.Errorf("Expected currentPos 0 after reset, got %d", tracker.currentPos)
	}
	if tracker.stopped {
		t.Error("Expected stopped to be false after reset")
	}
}

func TestLineTracker_UnlimitedMode(t *testing.T) {
	tracker, _ := NewLineTracker(0) // 0 means unlimited
	longText := strings.Repeat("Line\n", 100)
	display, overflow := tracker.Track(longText)
	if overflow {
		t.Error("Expected no overflow in unlimited mode")
	}
	if display != longText {
		t.Error("Expected full text to be displayed")
	}
}
