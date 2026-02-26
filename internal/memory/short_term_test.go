package memory

import (
	"testing"
	"time"
)

func TestShortTermMemory_AddSummary(t *testing.T) {
	tmpDir := t.TempDir()
	stm := NewShortTermMemory(tmpDir, 1000)

	summary := &Summary{
		ConversationID: "test-1",
		Summary:        "Test summary",
		Timestamp:      time.Now(),
		Tokens:         50,
	}

	err := stm.AddSummary(summary)
	if err != nil {
		t.Fatalf("AddSummary failed: %v", err)
	}

	summaries := stm.GetSummaries()
	if len(summaries) != 1 {
		t.Errorf("Expected 1 summary, got %d", len(summaries))
	}

	if summaries[0].ConversationID != "test-1" {
		t.Errorf("Expected conversation_id test-1, got %s", summaries[0].ConversationID)
	}
}

func TestShortTermMemory_TokenLimit(t *testing.T) {
	tmpDir := t.TempDir()
	maxTokens := 100
	stm := NewShortTermMemory(tmpDir, maxTokens)

	// Add summaries exceeding token limit
	for i := 0; i < 5; i++ {
		err := stm.AddSummary(&Summary{
			ConversationID: string(rune('a' + i)),
			Summary:        "Summary",
			Timestamp:      time.Now(),
			Tokens:         30, // 5 * 30 = 150 > 100
		})
		if err != nil {
			t.Fatalf("AddSummary failed: %v", err)
		}
	}

	summaries := stm.GetSummaries()
	totalTokens := 0
	for _, s := range summaries {
		totalTokens += s.Tokens
	}

	if totalTokens > maxTokens {
		t.Errorf("Total tokens %d exceeds limit %d", totalTokens, maxTokens)
	}
}

func TestShortTermMemory_LoadAndPersist(t *testing.T) {
	tmpDir := t.TempDir()
	stm := NewShortTermMemory(tmpDir, 1000)

	summary := &Summary{
		ConversationID: "persist-test",
		Summary:        "Persist this",
		Timestamp:      time.Now(),
		Tokens:         20,
	}

	err := stm.AddSummary(summary)
	if err != nil {
		t.Fatalf("AddSummary failed: %v", err)
	}

	// Create new instance to test loading
	stm2 := NewShortTermMemory(tmpDir, 1000)
	summaries := stm2.GetSummaries()

	if len(summaries) != 1 {
		t.Errorf("Expected 1 summary after reload, got %d", len(summaries))
	}

	if summaries[0].ConversationID != "persist-test" {
		t.Errorf("Expected conversation_id persist-test, got %s", summaries[0].ConversationID)
	}
}
