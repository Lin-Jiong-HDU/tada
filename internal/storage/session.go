package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

const (
	SessionDirName = "sessions"
	CurrentSession = "current.json"
	MaxHistory     = 100
)

// Session represents a conversation session
type Session struct {
	ID        string       `json:"id"`
	StartedAt time.Time    `json:"started_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Messages  []ai.Message `json:"messages"`
}

var currentSession *Session

// InitSession initializes or loads the current session
func InitSession() (*Session, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	sessionDir := filepath.Join(configDir, SessionDirName)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	sessionPath := filepath.Join(sessionDir, CurrentSession)

	// Try to load existing session
	if data, err := os.ReadFile(sessionPath); err == nil {
		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			return nil, fmt.Errorf("failed to unmarshal session: %w", err)
		}
		currentSession = &session
		return currentSession, nil
	}

	// Create new session
	session := Session{
		ID:        generateSessionID(),
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages:  []ai.Message{},
	}

	currentSession = &session
	return currentSession, nil
}

// GetCurrentSession returns the current session
func GetCurrentSession() *Session {
	return currentSession
}

// SaveSession saves the current session
func SaveSession() error {
	if currentSession == nil {
		return nil
	}

	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	sessionPath := filepath.Join(configDir, SessionDirName, CurrentSession)

	currentSession.UpdatedAt = time.Now()

	// Trim to max history
	if len(currentSession.Messages) > MaxHistory {
		currentSession.Messages = currentSession.Messages[len(currentSession.Messages)-MaxHistory:]
	}

	data, err := json.MarshalIndent(currentSession, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(sessionPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session: %w", err)
	}

	return nil
}

// AddMessage adds a message to the session
func AddMessage(role, content string) {
	if currentSession == nil {
		return
	}

	currentSession.Messages = append(currentSession.Messages, ai.Message{
		Role:    role,
		Content: content,
	})

	// Auto-save
	_ = SaveSession()
}

// ClearSession clears the current session
func ClearSession() error {
	if currentSession == nil {
		return nil
	}

	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	sessionPath := filepath.Join(configDir, SessionDirName, CurrentSession)

	if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	currentSession = nil
	return nil
}

func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().Unix())
}
