package ai

import "context"

// Message represents a chat message
type Message struct {
	Role    string `json:"role"` // "system" | "user" | "assistant"
	Content string `json:"content"`
}

// Intent represents the parsed user intent
type Intent struct {
	Commands     []Command `json:"commands"`
	Reason       string    `json:"reason"`
	NeedsConfirm bool      `json:"needs_confirm"`
}

// Command represents a shell command to execute
type Command struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
}

// AIProvider defines the interface for AI backends
type AIProvider interface {
	ParseIntent(ctx context.Context, input string, systemPrompt string) (*Intent, error)
	AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error)
	Chat(ctx context.Context, messages []Message) (string, error)
}
