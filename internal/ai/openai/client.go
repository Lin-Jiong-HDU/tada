package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

const (
	defaultSystemPrompt = `You are tada, a terminal AI assistant. Your job is to understand user requests and convert them into shell commands.

Rules:
1. Return ONLY valid JSON
2. For simple requests, return a single command
3. Explain your reasoning in the "reason" field
4. Mark dangerous commands (rm, chmod, etc.) with needs_confirm: true

Response format:
{
  "commands": [{"cmd": "command", "args": ["arg1", "arg2"]}],
  "reason": "explanation",
  "needs_confirm": false
}`
)

// Client implements AIProvider for OpenAI
type Client struct {
	apiKey  string
	model   string
	baseURL string
}

// NewClient creates a new OpenAI client
func NewClient(apiKey, model, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
	}
}

// ParseIntent parses user input and returns intent
func (c *Client) ParseIntent(ctx context.Context, input string, systemPrompt string) (*ai.Intent, error) {
	if systemPrompt == "" {
		systemPrompt = defaultSystemPrompt
	}

	prompt := fmt.Sprintf("User request: %s\n\nConvert this to shell commands. Return JSON only.", input)

	response, err := c.callAPI(ctx, []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, err
	}

	return c.parseIntentResponse(response)
}

// AnalyzeOutput analyzes command output
func (c *Client) AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error) {
	prompt := fmt.Sprintf("Command: %s\nOutput:\n%s\n\nBriefly explain what happened (max 2 sentences).", cmd, output)

	response, err := c.callAPI(ctx, []ai.Message{
		{Role: "system", Content: "You are a helpful assistant. Be brief and clear."},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return "", err
	}

	return response, nil
}

// Chat handles general conversation
func (c *Client) Chat(ctx context.Context, messages []ai.Message) (string, error) {
	return c.callAPI(ctx, messages)
}

// callAPI makes the actual API call
func (c *Client) callAPI(ctx context.Context, messages []ai.Message) (string, error) {
	// TODO: Implement actual HTTP call to OpenAI API
	// For now, return a mock response
	return "Mock response - implement HTTP client", nil
}

// parseIntentResponse parses JSON response into Intent
func (c *Client) parseIntentResponse(response string) (*ai.Intent, error) {
	var intent ai.Intent
	if err := json.Unmarshal([]byte(response), &intent); err != nil {
		return nil, fmt.Errorf("failed to parse intent: %w", err)
	}
	return &intent, nil
}
