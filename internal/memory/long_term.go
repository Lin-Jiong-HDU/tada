package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LongTermMemory manages user profile and entity tracking
type LongTermMemory struct {
	mu          sync.RWMutex
	storagePath string
	entityPath  string
	profilePath string
	threshold   int
	data        *LongTermMemoryData
}

// NewLongTermMemory creates a new long-term memory manager
func NewLongTermMemory(storagePath string, threshold int) *LongTermMemory {
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		// Continue in-memory only
	}

	ltm := &LongTermMemory{
		storagePath: storagePath,
		entityPath:  filepath.Join(storagePath, "entities.json"),
		profilePath: filepath.Join(storagePath, "user_profile.json"),
		threshold:   threshold,
		data: &LongTermMemoryData{
			Entities: make(map[string]*Entity),
			Profile:  &UserProfile{},
		},
	}

	ltm.load()
	return ltm
}

// load loads long-term memory from disk
func (l *LongTermMemory) load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Load entities
	if entityData, err := os.ReadFile(l.entityPath); err == nil {
		json.Unmarshal(entityData, &l.data.Entities)
	}

	// Load profile
	if profileData, err := os.ReadFile(l.profilePath); err == nil {
		json.Unmarshal(profileData, l.data.Profile)
	}

	return nil
}

// saveEntities saves entity data to disk
func (l *LongTermMemory) saveEntities() error {
	data, err := json.MarshalIndent(l.data.Entities, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.entityPath, data, 0644)
}

// saveProfile saves profile data to disk
func (l *LongTermMemory) saveProfile() error {
	data, err := json.MarshalIndent(l.data.Profile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.profilePath, data, 0644)
}

// UpdateEntity increments entity count and returns true if promoted
func (l *LongTermMemory) UpdateEntity(name string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	if entity, exists := l.data.Entities[name]; exists {
		entity.Count++
		entity.LastSeen = now

		if entity.Count >= l.threshold {
			// Promote to profile
			l.promoteEntityToProfile(name)
			return true, l.saveEntities()
		}
	} else {
		l.data.Entities[name] = &Entity{
			Count:     1,
			FirstSeen: now,
			LastSeen:  now,
		}
	}

	return false, l.saveEntities()
}

// promoteEntityToProfile adds entity to appropriate profile category
func (l *LongTermMemory) promoteEntityToProfile(name string) {
	// Simple heuristics for categorization
	// In production, this could use LLM classification
	profile := l.data.Profile

	// Check if it's a programming language
	languages := map[string]bool{
		"Go": true, "Python": true, "JavaScript": true,
		"TypeScript": true, "Rust": true, "Java": true,
	}
	if languages[name] {
		for _, lang := range profile.TechPreferences.Languages {
			if lang == name {
				return // Already exists
			}
		}
		profile.TechPreferences.Languages = append(profile.TechPreferences.Languages, name)
		return
	}

	// Check frameworks/libraries
	frameworks := map[string]bool{
		"React": true, "Vue": true, "Gin": true, "Echo": true,
	}
	if frameworks[name] {
		for _, fw := range profile.TechPreferences.Frameworks {
			if fw == name {
				return
			}
		}
		profile.TechPreferences.Frameworks = append(profile.TechPreferences.Frameworks, name)
	}

	// Default: add to work context as current project interest
	for _, proj := range profile.WorkContext.CurrentProjects {
		if proj == name {
			return
		}
	}
	profile.WorkContext.CurrentProjects = append(profile.WorkContext.CurrentProjects, name)
}

// UpdateProfile updates user profile from extraction results
func (l *LongTermMemory) UpdateProfile(extraction *ExtractionResult) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	profile := l.data.Profile

	// Update preferences from extraction
	for key, value := range extraction.Preferences {
		switch key {
		case "editor":
			profile.TechPreferences.Editors = appendUnique(profile.TechPreferences.Editors, value)
		case "timezone":
			profile.PersonalSettings.Timezone = value
		case "shell":
			profile.PersonalSettings.Shell = value
		case "communication_style":
			profile.BehaviorPatterns.PreferredCommunication = value
		}
	}

	// Track common topics as "often asks"
	for _, ctx := range extraction.Context {
		profile.BehaviorPatterns.OftenAsks = appendUnique(profile.BehaviorPatterns.OftenAsks, ctx)
	}

	return l.saveProfile()
}

// GetProfile returns the current user profile
func (l *LongTermMemory) GetProfile() *UserProfile {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	profileCopy := *l.data.Profile
	return &profileCopy
}

// GetEntityCount returns the current count for an entity
func (l *LongTermMemory) GetEntityCount(name string) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if entity, exists := l.data.Entities[name]; exists {
		return entity.Count
	}
	return 0
}

// appendUnique adds string to slice if not already present
func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}
