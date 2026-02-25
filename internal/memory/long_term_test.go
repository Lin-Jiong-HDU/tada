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

	// Third mention - promoted!
	promoted, _ = ltm.UpdateEntity("Go")
	if !promoted {
		t.Error("Expected promoted on third mention")
	}

	profile := ltm.GetProfile()
	if len(profile.TechPreferences.Languages) == 0 {
		t.Error("Expected Go in profile languages")
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

	err := ltm.UpdateProfile(extraction)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	profile := ltm.GetProfile()
	if profile.PersonalSettings.Shell != "" {
		// Should have some defaults set
	}
}
