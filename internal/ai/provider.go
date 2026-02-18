package ai

import "context"

// Message represents a chat message
type Message struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// Intent represents the parsed user intent
type Intent struct {
	Commands     []Command
	Reason       string
	NeedsConfirm bool
}

// Command represents a shell command to execute
type Command struct {
	Cmd  string
	Args []string
}

// AIProvider defines the interface for AI backends
type AIProvider interface {
	ParseIntent(ctx context.Context, input string, systemPrompt string) (*Intent, error)
	AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error)
	Chat(ctx context.Context, messages []Message) (string, error)
}
