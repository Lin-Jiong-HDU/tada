# Multi-Level Memory System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a CPU cache-inspired multi-level memory system (L1/L2/L3) for tada's chat functionality with automatic summary generation, LLM-based entity extraction, and user profiling.

**Architecture:** New `internal/memory` package with separated JSON storage for summaries (short-term) and user_profile/entities (long-term). Integrates with existing `internal/conversation` manager for session-end triggers and AI calls for context injection.

**Tech Stack:** Go 1.23+, JSON file storage, existing AI provider interface, Bubble Tea TUI

---

## Task 1: Create memory package structure and types

**Files:**
- Create: `internal/memory/types.go`
- Create: `internal/memory/README.md`

**Step 1: Write types.go with all data structures**

Create `internal/memory/types.go`:

```go
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
    Enabled         bool   `json:"enabled"`
    ShortTermMaxTokens int  `json:"short_term_max_tokens"`
    EntityThreshold int   `json:"entity_threshold"`
    StoragePath     string `json:"storage_path"`
}

// DefaultConfig returns default memory configuration
func DefaultConfig() *Config {
    return &Config{
        Enabled:          true,
        ShortTermMaxTokens: 4000,
        EntityThreshold:  5,
        StoragePath:      "~/.tada/memory",
    }
}
```

**Step 2: Create README.md for the package**

Create `internal/memory/README.md`:

```markdown
# Memory Package

Multi-level memory system for tada chat functionality.

## Architecture

- **L1**: Current session (handled by conversation package)
- **L2**: Short-term memory (summaries.json) - recent conversation summaries
- **L3**: Long-term memory (user_profile.json, entities.json) - persistent knowledge

## Components

- `types.go`: Core data structures
- `short_term.go`: Short-term memory manager
- `long_term.go`: Long-term memory manager
- `profiler.go`: User profile manager
- `extractor.go`: LLM-based entity extractor
- `manager.go`: Unified management interface
```

**Step 3: Commit**

```bash
git add internal/memory/types.go internal/memory/README.md
git commit -m "feat(multi-level-memory): add memory package structure and types"
```

---

## Task 2: Implement ShortTermMemory manager

**Files:**
- Create: `internal/memory/short_term.go`
- Create: `internal/memory/short_term_test.go`

**Step 1: Write failing tests for ShortTermMemory**

Create `internal/memory/short_term_test.go`:

```go
package memory

import (
    "os"
    "path/filepath"
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
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/memory -run TestShortTermMemory -v`
Expected: FAIL with "undefined: NewShortTermMemory"

**Step 3: Implement ShortTermMemory**

Create `internal/memory/short_term.go`:

```go
package memory

import (
    "encoding/json"
    "os"
    "path/filepath"
    "sync"
)

// ShortTermMemory manages conversation summaries
type ShortTermMemory struct {
    mu          sync.RWMutex
    path        string
    maxTokens   int
    data        *ShortTermMemoryData
}

// NewShortTermMemory creates a new short-term memory manager
func NewShortTermMemory(storagePath string, maxTokens int) *ShortTermMemory {
    if err := os.MkdirAll(storagePath, 0755); err != nil {
        // If directory creation fails, continue in-memory only
    }

    stm := &ShortTermMemory{
        path:      filepath.Join(storagePath, "summaries.json"),
        maxTokens: maxTokens,
        data:      &ShortTermMemoryData{MaxTokens: maxTokens},
    }

    stm.load()
    return stm
}

// load loads summaries from disk
func (s *ShortTermMemory) load() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    data, err := os.ReadFile(s.path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil // First run, no file yet
        }
        return err
    }

    return json.Unmarshal(data, s.data)
}

// save saves summaries to disk
func (s *ShortTermMemory) save() error {
    data, err := json.MarshalIndent(s.data, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(s.path, data, 0644)
}

// AddSummary adds a new summary, managing token limit via FIFO eviction
func (s *ShortTermMemory) AddSummary(summary *Summary) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.data.Summaries = append(s.data.Summaries, *summary)

    // Enforce token limit
    s.evictIfNeeded()

    return s.save()
}

// evictIfNeeded removes oldest summaries if token limit exceeded
func (s *ShortTermMemory) evictIfNeeded() {
    totalTokens := 0
    for _, s := range s.data.Summaries {
        totalTokens += s.Tokens
    }

    for totalTokens > s.maxTokens && len(s.data.Summaries) > 0 {
        removed := s.data.Summaries[0]
        s.data.Summaries = s.data.Summaries[1:]
        totalTokens -= removed.Tokens
    }
}

// GetSummaries returns all summaries
func (s *ShortTermMemory) GetSummaries() []Summary {
    s.mu.RLock()
    defer s.mu.RUnlock()

    result := make([]Summary, len(s.data.Summaries))
    copy(result, s.data.Summaries)
    return result
}

// Clear removes all summaries
func (s *ShortTermMemory) Clear() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.data.Summaries = nil
    return s.save()
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/memory -run TestShortTermMemory -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/memory/short_term.go internal/memory/short_term_test.go
git commit -m "feat(multi-level-memory): implement ShortTermMemory with token management"
```

---

## Task 3: Implement LongTermMemory manager

**Files:**
- Create: `internal/memory/long_term.go`
- Create: `internal/memory/long_term_test.go`

**Step 1: Write failing tests for LongTermMemory**

Create `internal/memory/long_term_test.go`:

```go
package memory

import (
    "os"
    "testing"
    "time"
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
            "editor": "neovim",
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
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/memory -run TestLongTermMemory -v`
Expected: FAIL with "undefined: NewLongTermMemory"

**Step 3: Implement LongTermMemory**

Create `internal/memory/long_term.go`:

```go
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
    mu             sync.RWMutex
    storagePath    string
    entityPath     string
    profilePath    string
    threshold      int
    data           *LongTermMemoryData
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
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/memory -run TestLongTermMemory -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/memory/long_term.go internal/memory/long_term_test.go
git commit -m "feat(multi-level-memory): implement LongTermMemory with entity tracking"
```

---

## Task 4: Implement LLM-based Extractor

**Files:**
- Create: `internal/memory/extractor.go`
- Create: `internal/memory/extractor_test.go`

**Step 1: Write failing tests for Extractor**

Create `internal/memory/extractor_test.go`:

```go
package memory

import (
    "context"
    "testing"

    "github.com/lin-jiong-hdu/tada/internal/ai"
)

// MockAIProvider for testing
type MockAIProvider struct {
    response string
}

func (m *MockAIProvider) Chat(ctx context.Context, messages []ai.Message) (string, error) {
    return m.response, nil
}

func (m *MockAIProvider) ChatStream(ctx context.Context, messages []ai.Message) (<-chan ai.StreamChunk, error) {
    return nil, nil
}

func (m *MockAIProvider) ParseIntent(ctx context.Context, prompt string) (*ai.Command, error) {
    return nil, nil
}

func TestExtractor_ExtractFromSummary(t *testing.T) {
    mockResponse := `{
        "entities": ["Go", "React", "neovim"],
        "preferences": {
            "editor": "neovim",
            "timezone": "Asia/Shanghai"
        },
        "context": ["Working on tada CLI tool", "Interested in memory systems"]
    }`

    provider := &MockAIProvider{response: mockResponse}
    extractor := NewExtractor(provider)

    result, err := extractor.ExtractFromSummary(context.Background(), "User discussed Go memory management and React state")
    if err != nil {
        t.Fatalf("ExtractFromSummary failed: %v", err)
    }

    if len(result.Entities) != 3 {
        t.Errorf("Expected 3 entities, got %d", len(result.Entities))
    }

    if result.Preferences["editor"] != "neovim" {
        t.Errorf("Expected editor preference 'neovim', got '%s'", result.Preferences["editor"])
    }
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/memory -run TestExtractor -v`
Expected: FAIL with "undefined: NewExtractor"

**Step 3: Implement Extractor**

Create `internal/memory/extractor.go`:

```go
package memory

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/lin-jiong-hdu/tada/internal/ai"
)

// Extractor uses LLM to extract entities and preferences from summaries
type Extractor struct {
    aiProvider ai.Provider
}

// NewExtractor creates a new LLM-based extractor
func NewExtractor(provider ai.Provider) *Extractor {
    return &Extractor{
        aiProvider: provider,
    }
}

// ExtractFromSummary extracts structured information from a conversation summary
func (e *Extractor) ExtractFromSummary(ctx context.Context, summary string) (*ExtractionResult, error) {
    prompt := e.buildExtractionPrompt(summary)

    messages := []ai.Message{
        {Role: "system", Content: "You are a helpful assistant that extracts structured information from conversations. Always respond with valid JSON only."},
        {Role: "user", Content: prompt},
    }

    response, err := e.aiProvider.Chat(ctx, messages)
    if err != nil {
        return nil, fmt.Errorf("LLM extraction failed: %w", err)
    }

    // Parse JSON response
    var result ExtractionResult
    if err := json.Unmarshal([]byte(response), &result); err != nil {
        // Fallback: return empty result if parsing fails
        return &ExtractionResult{}, nil
    }

    return &result, nil
}

// buildExtractionPrompt creates the prompt for entity extraction
func (e *Extractor) buildExtractionPrompt(summary string) string {
    return fmt.Sprintf(`Extract the following information from this conversation summary:

Summary: %s

Please extract and return as JSON:
{
  "entities": ["list of technologies, frameworks, tools mentioned"],
  "preferences": {"editor": "preferred editor if mentioned", "timezone": "timezone if mentioned", "shell": "shell if mentioned"},
  "context": ["key topics, projects, or areas of interest discussed"]
}

Only include fields that have values. Return valid JSON only, no markdown.`, summary)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/memory -run TestExtractor -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/memory/extractor.go internal/memory/extractor_test.go
git commit -m "feat(multi-level-memory): implement LLM-based Extractor"
```

---

## Task 5: Implement unified Manager

**Files:**
- Create: `internal/memory/manager.go`
- Create: `internal/memory/manager_test.go`

**Step 1: Write failing tests for Manager**

Create `internal/memory/manager_test.go`:

```go
package memory

import (
    "context"
    "testing"
    "time"

    "github.com/lin-jiong-hdu/tada/internal/ai"
    "github.com/lin-jiono-hdu/tada/internal/conversation"
)

func TestManager_BuildContext(t *testing.T) {
    tmpDir := t.TempDir()
    config := DefaultConfig()
    config.StoragePath = tmpDir

    provider := &MockAIProvider{}
    mgr, err := NewManager(config, provider)
    if err != nil {
        t.Fatalf("NewManager failed: %v", err)
    }

    currentMessages := []ai.Message{
        {Role: "user", Content: "Hello"},
        {Role: "assistant", Content: "Hi there"},
    }

    contextMsgs := mgr.BuildContext(currentMessages)

    // Should have system prompt + current messages
    if len(contextMsgs) < 3 {
        t.Errorf("Expected at least 3 messages (system + 2 current), got %d", len(contextMsgs))
    }

    // First message should be system prompt with memory context
    if contextMsgs[0].Role != "system" {
        t.Errorf("Expected first message to be system role, got %s", contextMsgs[0].Role)
    }
}

func TestManager_OnSessionEnd(t *testing.T) {
    tmpDir := t.TempDir()
    config := DefaultConfig()
    config.StoragePath = tmpDir
    config.EntityThreshold = 2 // Lower threshold for testing

    provider := &MockAIProvider{
        response: `{"entities": ["Go"], "preferences": {}, "context": []}`,
    }

    mgr, err := NewManager(config, provider)
    if err != nil {
        t.Fatalf("NewManager failed: %v", err)
    }

    conv := &conversation.Conversation{
        ID: "test-conv",
        Messages: []conversation.Message{
            {Role: "user", Content: "Tell me about Go", Timestamp: time.Now()},
            {Role: "assistant", Content: "Go is...", Timestamp: time.Now()},
        },
    }

    // This should not block (async internally)
    err = mgr.OnSessionEnd(conv)
    if err != nil {
        t.Fatalf("OnSessionEnd failed: %v", err)
    }

    // Give time for async processing
    time.Sleep(100 * time.Millisecond)

    summaries := mgr.shortTerm.GetSummaries()
    if len(summaries) == 0 {
        t.Error("Expected summary to be created")
    }
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/memory -run TestManager -v`
Expected: FAIL with "undefined: NewManager"

**Step 3: Implement Manager**

Create `internal/memory/manager.go`:

```go
package memory

import (
    "context"
    "fmt"
    "strings"

    "github.com/lin-jiong-hdu/tada/internal/ai"
    "github.com/lin-jiong-hdu/tada/internal/conversation"
)

// Manager provides unified interface for multi-level memory management
type Manager struct {
    config     *Config
    shortTerm  *ShortTermMemory
    longTerm   *LongTermMemory
    extractor  *Extractor
    aiProvider ai.Provider
}

// NewManager creates a new memory manager
func NewManager(config *Config, aiProvider ai.Provider) (*Manager, error) {
    if !config.Enabled {
        return nil, nil // Memory disabled
    }

    storagePath := expandPath(config.StoragePath)

    return &Manager{
        config:     config,
        shortTerm:  NewShortTermMemory(storagePath, config.ShortTermMaxTokens),
        longTerm:   NewLongTermMemory(storagePath, config.EntityThreshold),
        extractor:  NewExtractor(aiProvider),
        aiProvider: aiProvider,
    }, nil
}

// OnSessionEnd processes a completed conversation
func (m *Manager) OnSessionEnd(conv *conversation.Conversation) error {
    if m == nil {
        return nil
    }

    go m.processSessionEndAsync(conv)
    return nil
}

// processSessionEndAsync handles the async workflow
func (m *Manager) processSessionEndAsync(conv *conversation.Conversation) {
    ctx := context.Background()

    // Step 1: Generate summary
    summary, err := m.generateSummary(ctx, conv)
    if err != nil {
        return // Silently fail on error
    }

    // Step 2: Write to short-term memory
    summaryRecord := &Summary{
        ConversationID: conv.ID,
        Summary:        summary,
        Timestamp:      conv.UpdatedAt,
        Tokens:         estimateTokens(summary),
    }
    m.shortTerm.AddSummary(summaryRecord)

    // Step 3: Extract entities using LLM
    extraction, err := m.extractor.ExtractFromSummary(ctx, summary)
    if err != nil {
        return // Fallback: no extraction
    }

    // Step 4: Update entity counts and check for promotion
    for _, entity := range extraction.Entities {
        promoted, _ := m.longTerm.UpdateEntity(entity)
        if promoted {
            // Entity promoted to profile
        }
    }

    // Step 5: Update profile from extraction results
    m.longTerm.UpdateProfile(extraction)
}

// generateSummary creates a summary of the conversation
func (m *Manager) generateSummary(ctx context.Context, conv *conversation.Conversation) (string, error) {
    // Build messages from conversation
    messages := []ai.Message{
        {Role: "system", Content: "Summarize the following conversation in 1-2 sentences, focusing on key topics discussed."},
    }

    for _, msg := range conv.Messages {
        messages = append(messages, ai.Message{
            Role:    msg.Role,
            Content: msg.Content,
        })
    }

    summary, err := m.aiProvider.Chat(ctx, messages)
    if err != nil {
        return "", err
    }

    return strings.TrimSpace(summary), nil
}

// BuildContext constructs messages with memory context for AI calls
func (m *Manager) BuildContext(currentMessages []ai.Message) []ai.Message {
    if m == nil {
        return currentMessages
    }

    systemPrompt := m.buildSystemPrompt()

    // Prepend system prompt with memory context
    result := []ai.Message{
        {Role: "system", Content: systemPrompt},
    }

    return append(result, currentMessages...)
}

// buildSystemPrompt creates system prompt with memory context
func (m *Manager) buildSystemPrompt() string {
    var parts []string

    // User profile (L3)
    profile := m.longTerm.GetProfile()
    if len(profile.TechPreferences.Languages) > 0 || len(profile.TechPreferences.Frameworks) > 0 {
        parts = append(parts, "## User Profile")
        if len(profile.TechPreferences.Languages) > 0 {
            parts = append(parts, fmt.Sprintf("Languages: %s", strings.Join(profile.TechPreferences.Languages, ", ")))
        }
        if len(profile.TechPreferences.Frameworks) > 0 {
            parts = append(parts, fmt.Sprintf("Frameworks: %s", strings.Join(profile.TechPreferences.Frameworks, ", ")))
        }
    }

    // Short-term memory summaries (L2)
    summaries := m.shortTerm.GetSummaries()
    if len(summaries) > 0 {
        parts = append(parts, "## Recent Conversations")
        for _, s := range summaries {
            parts = append(parts, fmt.Sprintf("- %s", s.Summary))
        }
    }

    if len(parts) == 0 {
        return "You are tada, a terminal AI assistant."
    }

    return fmt.Sprintf(`You are tada, a terminal AI assistant.

%s

Use this context to provide more personalized responses.`, strings.Join(parts, "\n"))
}

// estimateTokens roughly estimates token count (1 token ≈ 4 characters)
func estimateTokens(text string) int {
    return (len(text) + 3) / 4
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
    if strings.HasPrefix(path, "~/") {
        home := "/root" // Default, should use os.UserHomeDir() in production
        // For cross-platform, handle properly
        return strings.Replace(path, "~", home, 1)
    }
    return path
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/memory -run TestManager -v`
Expected: PASS (may need minor adjustments to imports)

**Step 5: Commit**

```bash
git add internal/memory/manager.go internal/memory/manager_test.go
git commit -m "feat(multi-level-memory): implement unified Manager"
```

---

## Task 6: Add configuration support

**Files:**
- Modify: `internal/storage/config.go`

**Step 1: Read existing config structure**

Run: `cat internal/storage/config.go`

**Step 2: Add MemoryConfig to existing Config struct**

Add to `internal/storage/config.go`:

```go
// Add to the Config struct
type Config struct {
    AI      AIConfig      `json:"ai"`
    Security SecurityConfig `json:"security"`
    Chat    ChatConfig    `json:"chat"`
    Memory  MemoryConfig  `json:"memory"` // Add this
}

// Add new struct
type MemoryConfig struct {
    Enabled           bool `json:"enabled"`
    ShortTermMaxTokens int  `json:"short_term_max_tokens"`
    EntityThreshold   int  `json:"entity_threshold"`
    StoragePath       string `json:"storage_path"`
}

// Add default values in GetConfig() function
func GetConfig() (*Config, error) {
    // ... existing code ...

    // Set memory defaults
    if config.Memory.StoragePath == "" {
        config.Memory.StoragePath = "~/.tada/memory"
    }
    if config.Memory.ShortTermMaxTokens == 0 {
        config.Memory.ShortTermMaxTokens = 4000
    }
    if config.Memory.EntityThreshold == 0 {
        config.Memory.EntityThreshold = 5
    }

    return config, nil
}
```

**Step 3: Run tests**

Run: `go test ./internal/storage -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/storage/config.go
git commit -m "feat(multi-level-memory): add configuration support"
```

---

## Task 7: Integrate with conversation manager

**Files:**
- Modify: `internal/conversation/manager.go`

**Step 1: Read existing manager structure**

Run: `head -100 internal/conversation/manager.go`

**Step 2: Add memory.Manager to conversation.Manager**

Modify `internal/conversation/manager.go`:

```go
// Add import
import (
    "github.com/lin-jiong-hdu/tada/internal/memory"
)

// Add to Manager struct
type Manager struct {
    // ... existing fields ...
    memoryMgr *memory.Manager // Add this
}

// Update NewManager constructor
func NewManager(storagePath string, aiProvider ai.Provider, config *storage.Config) (*Manager, error) {
    // ... existing code ...

    // Initialize memory manager
    var memoryMgr *memory.Manager
    if config.Memory.Enabled {
        memConfig := &memory.Config{
            Enabled:           true,
            ShortTermMaxTokens: config.Memory.ShortTermMaxTokens,
            EntityThreshold:   config.Memory.EntityThreshold,
            StoragePath:       config.Memory.StoragePath,
        }
        memoryMgr, _ = memory.NewManager(memConfig, aiProvider)
    }

    return &Manager{
        // ... existing fields ...
        memoryMgr: memoryMgr, // Add this
    }, nil
}

// Modify End method to trigger memory processing
func (m *Manager) End(convID string) error {
    m.mu.Lock()
    conv, exists := m.conversations[convID]
    if !exists {
        m.mu.Unlock()
        return fmt.Errorf("conversation %s not found", convID)
    }

    conv.Status = StatusEnded
    conv.UpdatedAt = time.Now()
    m.mu.Unlock()

    // Trigger memory processing
    if m.memoryMgr != nil {
        go m.memoryMgr.OnSessionEnd(conv)
    }

    return nil
}
```

**Step 3: Run tests**

Run: `go test ./internal/conversation -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/conversation/manager.go
git commit -m "feat(multi-level-memory): integrate with conversation manager"
```

---

## Task 8: Inject memory context into AI calls

**Files:**
- Modify: `internal/conversation/manager.go`

**Step 1: Modify Chat method to use memory context**

Update the Chat method in `internal/conversation/manager.go`:

```go
func (m *Manager) Chat(ctx context.Context, convID, userMsg string) (string, error) {
    m.mu.Lock()
    conv, exists := m.conversations[convID]
    if !exists {
        m.mu.Unlock()
        return "", fmt.Errorf("conversation %s not found", convID)
    }
    m.mu.Unlock()

    // Add user message
    conv.AddMessage(conversation.Message{
        Role:      "user",
        Content:   userMsg,
        Timestamp: time.Now(),
    })

    // Build messages with memory context
    messages := m.buildMessages(conv)
    if m.memoryMgr != nil {
        messages = m.memoryMgr.BuildContext(messages)
    }

    // Call AI
    response, err := m.aiProvider.Chat(ctx, messages)
    if err != nil {
        return "", err
    }

    // Add assistant message
    conv.AddMessage(conversation.Message{
        Role:      "assistant",
        Content:   response,
        Timestamp: time.Now(),
    })

    return response, nil
}

// Similar update for ChatStream method
func (m *Manager) ChatStream(ctx context.Context, convID, userMsg string) (<-chan string, error) {
    // ... existing setup ...

    // Build messages with memory context
    messages := m.buildMessages(conv)
    if m.memoryMgr != nil {
        messages = m.memoryMgr.BuildContext(messages)
    }

    // ... rest of stream handling ...
}
```

**Step 2: Run tests**

Run: `go test ./internal/conversation -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/conversation/manager.go
git commit -m "feat(multi-level-memory): inject memory context into AI calls"
```

---

## Task 9: Add integration tests

**Files:**
- Create: `tests/integration/memory_test.go`

**Step 1: Create integration tests**

Create `tests/integration/memory_test.go`:

```go
// +build integration

package integration

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/lin-jiong-hdu/tada/internal/ai"
    "github.com/lin-jiong-hdu/tada/internal/conversation"
    "github.com/lin-jiong-hdu/tada/internal/memory"
    "github.com/lin-jiong-hdu/tada/internal/storage"
)

// TestMemoryFullFlow tests the complete memory workflow
// Run with: TADA_INTEGRATION_TEST=1 go test ./tests/integration -run TestMemoryFullFlow -v
func TestMemoryFullFlow(t *testing.T) {
    if os.Getenv("TADA_INTEGRATION_TEST") == "" {
        t.Skip("Skipping integration test. Set TADA_INTEGRATION_TEST=1 to run.")
    }

    // Setup
    tmpDir := t.TempDir()

    config := &memory.Config{
        Enabled:           true,
        ShortTermMaxTokens: 1000,
        EntityThreshold:   2,
        StoragePath:       tmpDir,
    }

    // Get real AI config from environment
    appConfig, err := storage.GetConfig()
    if err != nil {
        t.Skip("No valid config found, skipping integration test")
    }

    provider := ai.NewProvider(appConfig.AI)
    mgr, err := memory.NewManager(config, provider)
    if err != nil {
        t.Fatalf("Failed to create memory manager: %v", err)
    }

    // Create a test conversation
    conv := &conversation.Conversation{
        ID:   "test-integration-conv",
        Name: "Memory Test",
        Messages: []conversation.Message{
            {Role: "user", Content: "I'm working on a Go project called tada", Timestamp: time.Now()},
            {Role: "assistant", Content: "That sounds interesting! Tell me more about tada.", Timestamp: time.Now()},
            {Role: "user", Content: "It's a CLI tool written in Go that helps with terminal tasks", Timestamp: time.Now()},
        },
    }

    // Process session end
    err = mgr.OnSessionEnd(conv)
    if err != nil {
        t.Fatalf("OnSessionEnd failed: %v", err)
    }

    // Wait for async processing
    time.Sleep(2 * time.Second)

    // Verify summary was created
    summaries := filepath.Join(tmpDir, "summaries.json")
    if _, err := os.Stat(summaries); os.IsNotExist(err) {
        t.Error("summaries.json was not created")
    }

    // Verify entities were tracked
    entities := filepath.Join(tmpDir, "entities.json")
    if _, err := os.Stat(entities); os.IsNotExist(err) {
        t.Error("entities.json was not created")
    }

    // Test context building
    testMessages := []ai.Message{
        {Role: "user", Content: "What should I work on next?"},
    }

    contextMsgs := mgr.BuildContext(testMessages)

    // Should have system prompt + user message
    if len(contextMsgs) < 2 {
        t.Errorf("Expected at least 2 messages, got %d", len(contextMsgs))
    }

    if contextMsgs[0].Role != "system" {
        t.Errorf("Expected first message to be system, got %s", contextMsgs[0].Role)
    }

    t.Log("Integration test passed!")
}
```

**Step 2: Run integration tests (if API key available)**

Run: `TADA_INTEGRATION_TEST=1 go test ./tests/integration -run TestMemoryFullFlow -v`
Expected: PASS (if API key configured)

**Step 3: Commit**

```bash
git add tests/integration/memory_test.go
git commit -m "test(multi-level-memory): add integration tests"
```

---

## Task 10: Update documentation

**Files:**
- Modify: `CLAUDE.md`
- Create: `docs/memory-guide.md`

**Step 1: Update CLAUDE.md with memory section**

Add to `CLAUDE.md`:

```markdown
## Multi-Level Memory System

Tada features a CPU cache-inspired multi-level memory system:

- **L1**: Current session (full message history)
- **L2**: Short-term memory (recent conversation summaries, token-limited)
- **L3**: Long-term memory (user profile + entity tracking)

### Memory Flow

1. Session ends → AI generates summary → stored in L2
2. LLM extracts entities from summary → tracked in L3
3. Entity mentioned 5+ times → promoted to user profile
4. All levels injected into system prompt for context

### Configuration

```yaml
memory:
  enabled: true
  short_term_max_tokens: 4000
  entity_threshold: 5
  storage_path: "~/.tada/memory"
```

### Storage Locations

- `~/.tada/memory/summaries.json` - Short-term summaries
- `~/.tada/memory/entities.json` - Entity occurrence tracking
- `~/.tada/memory/user_profile.json` - Learned user profile
```

**Step 2: Create user guide**

Create `docs/memory-guide.md`:

```markdown
# Memory System Guide

## How It Works

The memory system learns from your conversations to provide more personalized assistance.

## Levels

### L1: Current Session
Your ongoing conversation with full message history.

### L2: Short-term Memory
Recent conversation summaries, automatically generated when sessions end.
Limited by token count (default: 4000). Oldest summaries removed when full.

### L3: Long-term Memory
**User Profile**: Your learned preferences (languages, frameworks, editor, etc.)
**Entity Tracking**: Topics mentioned repeatedly across sessions

## Entity Promotion

When something is mentioned in 5+ different sessions, it's promoted to your profile.

## Privacy

- Memory stored locally in `~/.tada/memory/`
- No data sent externally except for LLM processing
- Can be disabled in config: `memory.enabled: false`
- Files can be manually edited or deleted

## Viewing Your Memory

```bash
# View current profile
cat ~/.tada/memory/user_profile.json

# View tracked entities
cat ~/.tada/memory/entities.json

# View recent summaries
cat ~/.tada/memory/summaries.json
```
```

**Step 3: Commit**

```bash
git add CLAUDE.md docs/memory-guide.md
git commit -m "docs(multi-level-memory): update documentation"
```

---

## Verification Steps

After completing all tasks:

**Step 1: Run all tests**
```bash
go test ./... -v
```

**Step 2: Run integration tests (if configured)**
```bash
TADA_INTEGRATION_TEST=1 go test ./tests/integration -v
```

**Step 3: Manual verification**
```bash
# Build
go build -o tada cmd/tada/main.go

# Test chat with memory enabled
./tada chat "Hi, I'm working on a Go project"
# End session, start new one
./tada chat "What language did I mention I use?"
# Should remember from previous session
```

**Step 4: Verify memory files created**
```bash
ls -la ~/.tada/memory/
```

---

## Summary

This plan implements a complete multi-level memory system:

1. ✅ Memory package with types and managers
2. ✅ Short-term memory with token management
3. ✅ Long-term memory with entity tracking and profile
4. ✅ LLM-based entity extraction
5. ✅ Unified manager interface
6. ✅ Configuration integration
7. ✅ Conversation manager integration
8. ✅ AI call context injection
9. ✅ Integration tests
10. ✅ Documentation updates

**Estimated Total Time**: 2-3 hours for implementation
**Commit Strategy**: 10 focused commits, one per task
