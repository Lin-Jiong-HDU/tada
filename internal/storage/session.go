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

	sessionDir, err := getSessionDir(currentSession.ID)
	if err != nil {
		return err
	}

	sessionPath := filepath.Join(sessionDir, "session.json")

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

	sessionDir, err := getSessionDir(currentSession.ID)
	if err != nil {
		return err
	}

	sessionPath := filepath.Join(sessionDir, "session.json")

	if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	currentSession = nil
	return nil
}

func generateSessionID() string {
	now := time.Now()
	return fmt.Sprintf("%d-%02d-%02d-%02d%02d%02d-%09d",
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		now.Second(),
		now.Nanosecond())
}

func getSessionDir(sessionID string) (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	sessionDir := filepath.Join(configDir, SessionDirName, sessionID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create session directory: %w", err)
	}
	return sessionDir, nil
}
