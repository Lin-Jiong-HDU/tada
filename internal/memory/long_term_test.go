package memory

import (
	"testing"
)

func TestLongTermMemory_UpdateEntity(t *testing.T) {
	tmpDir := t.TempDir()
	ltm := NewLongTermMemory(tmpDir, 3) // threshold = 3

	// First mention - not promoted
	promoted, err := ltm.UpdateEntity("Go")
	if err != nil {
		t.Fatalf("UpdateEntity failed: %v", err)
	}
	if promoted {
		t.Error("Expected not promoted on first mention")
	}

	// Second mention - not promoted
	promoted, _ = ltm.UpdateEntity("Go")
	if promoted {
		t.Error("Expected not promoted on second mention")
	}

	// Third mention - threshold reached!
	promoted, _ = ltm.UpdateEntity("Go")
	if !promoted {
		t.Error("Expected promoted on third mention")
	}

	// Verify entity count
	if ltm.GetEntityCount("Go") != 3 {
		t.Error("Expected entity count to be 3")
	}
}

func TestLongTermMemory_UpdateProfile(t *testing.T) {
	tmpDir := t.TempDir()
	ltm := NewLongTermMemory(tmpDir, 5)

	extraction := &ExtractionResult{
		Entities: []string{"Go", "React"},
		Preferences: map[string]string{
			"editor":   "neovim",
			"timezone": "Asia/Shanghai",
		},
		Context: []string{"Working on tada project"},
	}

	// UpdateProfile is deprecated and does nothing
	err := ltm.UpdateProfile(extraction)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	// Verify profile markdown is initially empty
	if ltm.GetProfileMarkdown() != "" {
		t.Error("Expected empty profile markdown initially")
	}
}
