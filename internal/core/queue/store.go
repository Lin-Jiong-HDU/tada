package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// QueueFile represents the persisted queue data
type QueueFile struct {
	Tasks []*Task `json:"tasks"`
}

// Store handles JSON persistence of the task queue
type Store struct {
	filePath string
}

// NewStore creates a new store for the given file path
func NewStore(filePath string) *Store {
	return &Store{filePath: filePath}
}

// Save persists tasks to JSON file
func (s *Store) Save(tasks []*Task) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	queueFile := QueueFile{Tasks: tasks}

	data, err := json.MarshalIndent(queueFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal queue: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write queue file: %w", err)
	}

	return nil
}

// Load loads tasks from JSON file
func (s *Store) Load() ([]*Task, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No queue file exists yet
			return []*Task{}, nil
		}
		return nil, fmt.Errorf("failed to read queue file: %w", err)
	}

	var queueFile QueueFile
	if err := json.Unmarshal(data, &queueFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal queue: %w", err)
	}

	return queueFile.Tasks, nil
}
