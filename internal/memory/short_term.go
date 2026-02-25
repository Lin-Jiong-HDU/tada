package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// ShortTermMemory manages conversation summaries
type ShortTermMemory struct {
	mu        sync.RWMutex
	path      string
	maxTokens int
	data      *ShortTermMemoryData
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
