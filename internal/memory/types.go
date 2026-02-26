package memory

import "time"

// Summary represents a single conversation summary in short-term memory
type Summary struct {
	ConversationID string    `json:"conversation_id"`
	Summary        string    `json:"summary"`
	Timestamp      time.Time `json:"timestamp"`
	Tokens         int       `json:"tokens"`
}

// ShortTermMemoryData holds the summaries.json structure
type ShortTermMemoryData struct {
	MaxTokens int       `json:"max_tokens"`
	Summaries []Summary `json:"summaries"`
}

// Entity tracks occurrences of a mentioned entity
type Entity struct {
	Count     int       `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// EntityData holds the entities.json structure
type EntityData struct {
	Entities map[string]*Entity `json:"entities"`
}

// TechPreferences tracks user's technical preferences
type TechPreferences struct {
	Languages  []string `json:"languages"`
	Frameworks []string `json:"frameworks"`
	Editors    []string `json:"editors"`
}

// WorkContext tracks user's work context
type WorkContext struct {
	CurrentProjects []string `json:"current_projects"`
	CommonPaths     []string `json:"common_paths"`
}

// BehaviorPatterns tracks user's behavior patterns
type BehaviorPatterns struct {
	PreferredCommunication string   `json:"preferred_communication"`
	OftenAsks              []string `json:"often_asks"`
}

// PersonalSettings tracks user's personal settings
type PersonalSettings struct {
	Timezone string `json:"timezone"`
	Shell    string `json:"shell"`
}

// UserProfile represents the user profile in long-term memory
type UserProfile struct {
	TechPreferences  TechPreferences  `json:"tech_preferences"`
	WorkContext      WorkContext      `json:"work_context"`
	BehaviorPatterns BehaviorPatterns `json:"behavior_patterns"`
	PersonalSettings PersonalSettings `json:"personal_settings"`
}

// LongTermMemoryData holds combined long-term memory
type LongTermMemoryData struct {
	Entities map[string]*Entity `json:"entities"`
	Profile  *UserProfile       `json:"profile"`
}

// ExtractionResult represents LLM extraction output
type ExtractionResult struct {
	Entities    []string          `json:"entities"`
	Preferences map[string]string `json:"preferences"`
	Context     []string          `json:"context"`
}

// Config holds memory configuration
type Config struct {
	Enabled            bool   `json:"enabled"`
	ShortTermMaxTokens int    `json:"short_term_max_tokens"`
	EntityThreshold    int    `json:"entity_threshold"`
	StoragePath        string `json:"storage_path"`
}

// DefaultConfig returns default memory configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:            true,
		ShortTermMaxTokens: 4000,
		EntityThreshold:    5,
		StoragePath:        "~/.tada/memory",
	}
}
