package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

// LongTermMemory manages user profile and entity tracking
type LongTermMemory struct {
	mu           sync.RWMutex
	storagePath  string
	entityPath   string
	profilePath  string
	threshold    int
	entities     map[string]*Entity
	profileMD    string
	aiProvider   ai.AIProvider
	promptLoader *PromptLoader
}

// NewLongTermMemory creates a new long-term memory manager
func NewLongTermMemory(storagePath string, threshold int) *LongTermMemory {
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		// Continue in-memory only
	}

	ltm := &LongTermMemory{
		storagePath: storagePath,
		entityPath:  filepath.Join(storagePath, "entities.json"),
		profilePath: filepath.Join(storagePath, "user_profile.md"),
		threshold:   threshold,
		entities:    make(map[string]*Entity),
		profileMD:   "",
	}

	ltm.load()
	return ltm
}

// SetAIProvider sets the AI provider for profile updates
func (l *LongTermMemory) SetAIProvider(provider ai.AIProvider, promptLoader *PromptLoader) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.aiProvider = provider
	l.promptLoader = promptLoader
}

// load loads long-term memory from disk
func (l *LongTermMemory) load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Load entities
	if entityData, err := os.ReadFile(l.entityPath); err == nil {
		json.Unmarshal(entityData, &l.entities)
	}

	// Load profile markdown directly
	if profileData, err := os.ReadFile(l.profilePath); err == nil {
		l.profileMD = string(profileData)
	}

	return nil
}

// saveEntities saves entity data to disk
func (l *LongTermMemory) saveEntities() error {
	data, err := json.MarshalIndent(l.entities, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.entityPath, data, 0644)
}

// saveProfile saves profile markdown to file
func (l *LongTermMemory) saveProfile() error {
	return os.WriteFile(l.profilePath, []byte(l.profileMD), 0644)
}

// UpdateProfileWithLLM updates user profile markdown using LLM
func (l *LongTermMemory) UpdateProfileWithLLM(ctx context.Context, newInfo string) error {
	// Phase 1: read-only access under read lock
	l.mu.RLock()
	if l.aiProvider == nil {
		l.mu.RUnlock()
		return fmt.Errorf("AI provider not set")
	}
	provider := l.aiProvider
	currentProfile := l.profileMD
	l.mu.RUnlock()

	// Build update prompt with current markdown and new info
	updatePrompt := l.buildProfileUpdatePrompt(currentProfile, newInfo)

	messages := []ai.Message{
		{Role: "system", Content: "You are a helpful assistant that maintains user profiles. Always respond with valid markdown only."},
		{Role: "user", Content: updatePrompt},
	}

	// Perform the potentially long-running LLM call without holding the lock
	response, err := provider.Chat(ctx, messages)
	if err != nil {
		return fmt.Errorf("LLM profile update failed: %w", err)
	}

	updatedProfile := strings.TrimSpace(response)

	// Phase 2: acquire write lock to commit the updated profile and persist
	l.mu.Lock()
	defer l.mu.Unlock()

	l.profileMD = updatedProfile
	return l.saveProfile()
}

// buildProfileUpdatePrompt creates prompt for profile update
func (l *LongTermMemory) buildProfileUpdatePrompt(currentMD, newInfo string) string {
	// Try to load template, or use default
	defaultPrompt := `Update the following user profile with the new information provided.

## Current Profile
{{profile}}

## New Information
{{info}}

Please update the profile by:
1. Adding any new information that should be included
2. Updating existing information if the new information contradicts or refines it
3. Removing information that is no longer relevant
4. Keeping the same markdown format

Return ONLY the updated profile markdown, no other text.`

	var updatePrompt string
	if l.promptLoader != nil {
		template, err := l.promptLoader.Load("update-profile")
		if err == nil {
			updatePrompt = template.SystemPrompt
			updatePrompt = strings.ReplaceAll(updatePrompt, "{{profile}}", currentMD)
			updatePrompt = strings.ReplaceAll(updatePrompt, "{{info}}", newInfo)
			return updatePrompt
		}
	}

	// Fallback to default
	return fmt.Sprintf(strings.ReplaceAll(strings.ReplaceAll(defaultPrompt, "{{profile}}", "%s"), "{{info}}", "%s"), currentMD, newInfo)
}

// UpdateEntity increments entity count and returns true if threshold reached
func (l *LongTermMemory) UpdateEntity(name string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	if entity, exists := l.entities[name]; exists {
		entity.Count++
		entity.LastSeen = now

		if entity.Count >= l.threshold {
			// Threshold reached - caller can trigger profile update
			return true, l.saveEntities()
		}
	} else {
		l.entities[name] = &Entity{
			Count:     1,
			FirstSeen: now,
			LastSeen:  now,
		}
	}

	return false, l.saveEntities()
}

// promoteEntityToProfile adds entity to appropriate profile category
func (l *LongTermMemory) promoteEntityToProfile(name string) {
	// Entity promotion is now handled by LLM in UpdateProfileWithLLM
	// This method is kept for compatibility but does nothing
}

// UpdateProfile updates user profile from extraction results (deprecated - use UpdateProfileWithLLM)
func (l *LongTermMemory) UpdateProfile(extraction *ExtractionResult) error {
	// This method is deprecated - profile updates are now handled by LLM
	return nil
}

// GetProfileMarkdown returns the current user profile markdown
func (l *LongTermMemory) GetProfileMarkdown() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.profileMD
}

// GetProfile returns the current user profile (deprecated - use GetProfileMarkdown)
func (l *LongTermMemory) GetProfile() *UserProfile {
	l.mu.RLock()
	defer l.mu.RUnlock()
	// Return empty profile for compatibility
	return &UserProfile{}
}

// GetEntityCount returns the current count for an entity
func (l *LongTermMemory) GetEntityCount(name string) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if entity, exists := l.entities[name]; exists {
		return entity.Count
	}
	return 0
}
