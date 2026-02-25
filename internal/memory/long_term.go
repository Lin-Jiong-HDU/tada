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
	mu          sync.RWMutex
	storagePath string
	entityPath  string
	profilePath string
	threshold   int
	data        *LongTermMemoryData
	aiProvider  ai.AIProvider
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
		data: &LongTermMemoryData{
			Entities: make(map[string]*Entity),
			Profile:  &UserProfile{},
		},
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
		json.Unmarshal(entityData, &l.data.Entities)
	}

	// Load profile from markdown
	if profileData, err := os.ReadFile(l.profilePath); err == nil {
		profile, err := l.parseProfileFromMarkdown(string(profileData))
		if err == nil {
			l.data.Profile = profile
		}
		// If parsing fails, use empty profile
	}

	return nil
}

// parseProfileFromMarkdown parses UserProfile from markdown content
func (l *LongTermMemory) parseProfileFromMarkdown(content string) (*UserProfile, error) {
	profile := &UserProfile{}

	lines := strings.Split(content, "\n")
	var currentSection string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and headers
		if line == "" || strings.HasPrefix(line, "#") {
			if strings.Contains(line, "Technical Preferences") || strings.Contains(line, "技术偏好") {
				currentSection = "tech"
			} else if strings.Contains(line, "Work Context") || strings.Contains(line, "工作背景") {
				currentSection = "work"
			} else if strings.Contains(line, "Behavior Patterns") || strings.Contains(line, "行为模式") {
				currentSection = "behavior"
			} else if strings.Contains(line, "Personal Settings") || strings.Contains(line, "个人设置") {
				currentSection = "personal"
			}
			continue
		}

		// Parse list items
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			item := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "-"), "*"))

			// Extract key-value pairs
			if strings.Contains(item, ":") {
				parts := strings.SplitN(item, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])

					switch currentSection {
					case "tech":
						if key == "Languages" || key == "语言" {
							profile.TechPreferences.Languages = parseList(value)
						} else if key == "Frameworks" || key == "框架" {
							profile.TechPreferences.Frameworks = parseList(value)
						} else if key == "Editors" || key == "编辑器" {
							profile.TechPreferences.Editors = parseList(value)
						}
					case "work":
						if key == "Projects" || key == "项目" {
							profile.WorkContext.CurrentProjects = parseList(value)
						} else if key == "Common Paths" || key == "常用路径" {
							profile.WorkContext.CommonPaths = parseList(value)
						}
					case "behavior":
						if key == "Communication Style" || key == "沟通方式" {
							profile.BehaviorPatterns.PreferredCommunication = value
						} else if key == "Often Asks" || key == "常问问题" {
							profile.BehaviorPatterns.OftenAsks = parseList(value)
						}
					case "personal":
						if key == "Timezone" || key == "时区" {
							profile.PersonalSettings.Timezone = value
						} else if key == "Shell" || key == "命令行" {
							profile.PersonalSettings.Shell = value
						}
					}
				}
			}
		}
	}

	return profile, nil
}

// parseList parses a comma-separated list
func parseList(s string) []string {
	items := strings.Split(s, ",")
	var result []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

// saveEntities saves entity data to disk
func (l *LongTermMemory) saveEntities() error {
	data, err := json.MarshalIndent(l.data.Entities, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.entityPath, data, 0644)
}

// saveProfile saves profile data to markdown file
func (l *LongTermMemory) saveProfile() error {
	content := l.formatProfileAsMarkdown()
	return os.WriteFile(l.profilePath, []byte(content), 0644)
}

// formatProfileAsMarkdown formats UserProfile as markdown
func (l *LongTermMemory) formatProfileAsMarkdown() string {
	profile := l.data.Profile
	var sb strings.Builder

	sb.WriteString("# User Profile\n\n")

	// Technical Preferences
	sb.WriteString("## Technical Preferences\n\n")
	if len(profile.TechPreferences.Languages) > 0 {
		sb.WriteString(fmt.Sprintf("- Languages: %s\n", strings.Join(profile.TechPreferences.Languages, ", ")))
	}
	if len(profile.TechPreferences.Frameworks) > 0 {
		sb.WriteString(fmt.Sprintf("- Frameworks: %s\n", strings.Join(profile.TechPreferences.Frameworks, ", ")))
	}
	if len(profile.TechPreferences.Editors) > 0 {
		sb.WriteString(fmt.Sprintf("- Editors: %s\n", strings.Join(profile.TechPreferences.Editors, ", ")))
	}
	sb.WriteString("\n")

	// Work Context
	if len(profile.WorkContext.CurrentProjects) > 0 || len(profile.WorkContext.CommonPaths) > 0 {
		sb.WriteString("## Work Context\n\n")
		if len(profile.WorkContext.CurrentProjects) > 0 {
			sb.WriteString(fmt.Sprintf("- Projects: %s\n", strings.Join(profile.WorkContext.CurrentProjects, ", ")))
		}
		if len(profile.WorkContext.CommonPaths) > 0 {
			sb.WriteString(fmt.Sprintf("- Common Paths: %s\n", strings.Join(profile.WorkContext.CommonPaths, ", ")))
		}
		sb.WriteString("\n")
	}

	// Behavior Patterns
	if profile.BehaviorPatterns.PreferredCommunication != "" || len(profile.BehaviorPatterns.OftenAsks) > 0 {
		sb.WriteString("## Behavior Patterns\n\n")
		if profile.BehaviorPatterns.PreferredCommunication != "" {
			sb.WriteString(fmt.Sprintf("- Communication Style: %s\n", profile.BehaviorPatterns.PreferredCommunication))
		}
		if len(profile.BehaviorPatterns.OftenAsks) > 0 {
			sb.WriteString(fmt.Sprintf("- Often Asks: %s\n", strings.Join(profile.BehaviorPatterns.OftenAsks, ", ")))
		}
		sb.WriteString("\n")
	}

	// Personal Settings
	if profile.PersonalSettings.Timezone != "" || profile.PersonalSettings.Shell != "" {
		sb.WriteString("## Personal Settings\n\n")
		if profile.PersonalSettings.Timezone != "" {
			sb.WriteString(fmt.Sprintf("- Timezone: %s\n", profile.PersonalSettings.Timezone))
		}
		if profile.PersonalSettings.Shell != "" {
			sb.WriteString(fmt.Sprintf("- Shell: %s\n", profile.PersonalSettings.Shell))
		}
		sb.WriteString("\n")
	}

	// Add metadata
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("*Last updated: %s*\n", time.Now().Format("2006-01-02 15:04:05")))

	return sb.String()
}

// UpdateProfileWithLLM updates user profile using LLM
func (l *LongTermMemory) UpdateProfileWithLLM(ctx context.Context, newInfo string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.aiProvider == nil {
		return fmt.Errorf("AI provider not set")
	}

	// Get current profile as markdown
	currentProfile := l.formatProfileAsMarkdown()

	// Build update prompt
	updatePrompt := l.buildProfileUpdatePrompt(currentProfile, newInfo)

	messages := []ai.Message{
		{Role: "system", Content: "You are a helpful assistant that maintains user profiles. Always respond with valid markdown only."},
		{Role: "user", Content: updatePrompt},
	}

	response, err := l.aiProvider.Chat(ctx, messages)
	if err != nil {
		return fmt.Errorf("LLM profile update failed: %w", err)
	}

	// Parse updated profile
	updatedProfile, err := l.parseProfileFromMarkdown(response)
	if err != nil {
		return fmt.Errorf("failed to parse updated profile: %w", err)
	}

	l.data.Profile = updatedProfile
	return l.saveProfile()
}

// buildProfileUpdatePrompt creates prompt for profile update
func (l *LongTermMemory) buildProfileUpdatePrompt(currentProfile, newInfo string) string {
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
			updatePrompt = strings.ReplaceAll(updatePrompt, "{{profile}}", currentProfile)
			updatePrompt = strings.ReplaceAll(updatePrompt, "{{info}}", newInfo)
			return updatePrompt
		}
	}

	// Fallback to default
	return fmt.Sprintf(strings.ReplaceAll(strings.ReplaceAll(defaultPrompt, "{{profile}}", "%s"), "{{info}}", "%s"), currentProfile, newInfo)
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
